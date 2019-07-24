package app

import (
	"errors"
	"github.com/nothollyhigh/kiss/log"
	tnet "github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTimeout  = time.Second * 5
	unreachableTime = time.Duration(-1)
	unpausecheck    = time.Second * 0
	COUNT_MINUTES   = 60
)

type FailedInMunite struct {
	Time      time.Time
	FailedNum int64
}

var (
	ErrorInvalidAddr = errors.New("Invalid Addr")
)

/* 线路 */
type Line struct {
	sync.RWMutex
	Running  bool          /* 线路检测是否在进行的标志 */
	Born     time.Time     /* 线路出生时间 */
	Remote   string        /* 线路指向的服务器地址 */
	Delay    time.Duration /* 线路延迟 */
	Timeout  time.Duration /* 进行线路检测时的超时时间 */
	Interval time.Duration /* 线路检测的时间周期 */
	Timer    *time.Timer   /* 用于定时进行线路检测的定时器 */
	CurLoad  int64         /* 当前线路负载 */
	MaxLoad  int64         /* 线路最大负载 */

	IsPaused bool /* 线路暂停使用的标志 */
	Redirect bool /* 线路是否需要向服务器发送客户端真实IP的标志 */

	ChUpdateDelay chan time.Duration /* 用于外部更新线路当前延迟的channel，外部进行更新后本线路的线路检测reset计时周期避免浪费 */

	FailedRecord     [COUNT_MINUTES]FailedInMunite /* 环形队列，记录过去COUNT_MINUTES分钟内连接失败次数 */
	FailedRecordHead int                           /* 环形队列头 */
}

/* 线路分数，小于0为线路不可用 */
func (line *Line) Score() int64 {
	if !line.IsPaused && line.Delay != unreachableTime && line.CurLoad < line.MaxLoad {
		return (line.MaxLoad - line.CurLoad)
	}
	return -1
}

func (line *Line) CheckLine(now time.Time) {
	if err := tnet.Ping(line.Remote, line.Timeout); err != nil {
		line.Delay = unreachableTime //line.Timeout
		log.Error("CheckLine (Addr: %s) Failed, err: %v", line.Remote, err)
		return
	} else {
		line.Delay = time.Since(now)
	}

	log.Info("CheckLine (Addr: %s, Delay: %v)", line.Remote, line.Delay)
}

/* 启动线路检测 */
func (line *Line) Start(idx int) {
	line.Lock()
	defer line.Unlock()
	if line.Running {
		return
	}

	line.Running = true

	util.Go(func() {
		line.Born = time.Now()
		line.Timer = time.NewTimer(line.Interval)

		line.CheckLine(line.Born)
		for {
			select {
			case now, ok := <-line.Timer.C:
				if !ok {
					return
				}
				line.CheckLine(now)
			case delay, ok := <-line.ChUpdateDelay:
				if !ok {
					return
				}
				line.Delay = delay
			}
			line.Timer.Reset(line.Interval)

		}
	})
}

/* 更新线路延迟 */
func (line *Line) UpdateDelay(delay time.Duration) {
	line.RLock()
	defer line.RUnlock()
	if !line.Running {
		return
	}
	select {
	case line.ChUpdateDelay <- delay:
	default:
	}
}

/* 停止线路检测 */
func (line *Line) Stop() {
	line.Lock()
	defer line.Unlock()
	if !line.Running {
		return
	}

	line.Running = false

	line.Timer.Stop()
	close(line.ChUpdateDelay)
}

/* 更新负载 */
func (line *Line) UpdateLoad(delta int64) {
	atomic.AddInt64(&(line.CurLoad), delta)
}

/* 暂停在此线路选路和进行代理连接 */
func (line *Line) Pause() {
	line.IsPaused = true
}

/* 恢复在此线路选路和进行代理连接 */
func (line *Line) UnPause() {
	line.IsPaused = false

	line.CheckLine(time.Now())
}

/* 根据线路配置决定是否向服务器发送重定向包，应在刚建立与服务器的连接时首先进行然后再发送其他包 */
func (line *Line) HandleRedirect(conn *net.TCPConn, addr string) error {
	if line.Redirect {
		var err error

		log.Info("HandleRedirect (%s -> %s)", addr, conn.RemoteAddr().String())

		addrstr := strings.Split(addr, ":")[0]
		msg := tnet.RealIpMsg(addrstr)

		if err = conn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
			return err
		}
		_, err = conn.Write(msg.Data())

		return err
	}

	return nil
}

/* 更新为客户端与服务器建立连接的失败总次数，以及近期每分钟内的失败次数记录 */
func (line *Line) UpdateFailedNum(delta int64) {
	line.Lock()
	defer line.Unlock()

	currHead := int(time.Since(bornTime).Minutes()) % COUNT_MINUTES
	if currHead != line.FailedRecordHead || time.Since(line.FailedRecord[line.FailedRecordHead].Time).Minutes() >= 1.0 {
		line.FailedRecordHead = currHead
		line.FailedRecord[currHead] = FailedInMunite{
			Time:      time.Now(),
			FailedNum: 1,
		}
	} else {
		line.FailedRecord[currHead].FailedNum++
	}
}

/* 获取近期n分钟内为客户端与服务器建立连接的失败次数 */
func (line *Line) GetFailedInLastNMinutes(n int) int64 {
	line.Lock()
	defer line.Unlock()

	if n > 0 && n <= COUNT_MINUTES {
		var total int64 = 0
		for i := 0; i < n; i++ {
			if time.Since(line.FailedRecord[(line.FailedRecordHead+COUNT_MINUTES-i)%COUNT_MINUTES].Time).Minutes() >= float64(n) {
				break
			}
			total += line.FailedRecord[(line.FailedRecordHead+COUNT_MINUTES-i)%COUNT_MINUTES].FailedNum
		}
		return total
	}
	return -1
}

/* 创建新线路 */
func NewLine(remote string, timeout time.Duration, interval time.Duration, maxLoad int64, redirect bool) *Line {
	line := &Line{
		Remote:        remote,
		Delay:         unreachableTime,
		Timeout:       timeout,
		Interval:      interval,
		Timer:         nil,
		CurLoad:       0,
		MaxLoad:       maxLoad,
		IsPaused:      false,
		Redirect:      redirect,
		ChUpdateDelay: make(chan time.Duration, 1024),
	}
	failed := FailedInMunite{
		Time: time.Now(),
	}
	for i := 0; i < COUNT_MINUTES; i++ {
		line.FailedRecord[i] = failed
	}
	return line
}
