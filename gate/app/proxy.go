package app

import (
	"sync"
	"time"
)

var (
	PT_TCP       = "tcp"
	PT_WEBSOCKET = "websocket"

	/* 默认配置项 */
	DEFAULT_TCP_NODELAY            = true              /* tcp nodelay */
	DEFAULT_TCP_REDIRECT           = false             /* 是否向服务器发送客户端IP */
	DEFAULT_TCP_HEARTBEAT          = time.Second * 30  /* tcp代理设置的心跳时间 */
	DEFAULT_TCP_KEEPALIVE_INTERVAL = time.Second * 600 /* tcp代理设置的 keepalive 时间 */
	DEFAULT_TCP_READ_BUF_LEN       = 1024 * 4          /* tcp 接收缓冲区 */
	DEFAULT_TCP_WRITE_BUF_LEN      = 1024 * 4          /* tcp 发送缓冲区 */
	DEFAULT_TCP_READ_BLOCK_TIME    = time.Second * 35  /* tcp 读数据超时时间 */
	DEFAULT_TCP_WRITE_BLOCK_TIME   = time.Second * 5   /* tcp 写数据超时时间 */
	DEFAULT_TCP_CHECKLINE_INTERVAL = time.Second * 60  /* 线路检测周期 */
	DEFAULT_TCP_CHECKLINE_TIMEOUT  = time.Second * 10  /* 线路检测超时时间 */
)

type IProxy interface {
	GetBestLine() *Line
}

/* 每个 ProxyBase 管理一组 Line ，Proxy is a ProxyBase */
type ProxyBase struct {
	sync.RWMutex

	name  string
	ptype string
	local string
	lines []*Line
}

/* 当前最适合的线路 */
func (mgr *ProxyBase) GetBestLine() *Line {
	mgr.RLock()
	defer mgr.RUnlock()

	return mgr.GetBestLineWithoutLock()
}

/* 当前最适合的线路 */
func (mgr *ProxyBase) GetBestLineWithoutLock() *Line {
	if len(mgr.lines) > 0 {
		line := mgr.lines[0]
		for i := 1; i < len(mgr.lines); i++ {
			if mgr.lines[i].Score() > line.Score() {
				line = mgr.lines[i]
			}
		}
		if line.Score() >= 0 {
			return line
		}
	}

	return nil
}

/* 添加一个 Line */
func (mgr *ProxyBase) AddLine(addr string, timeout time.Duration, interval time.Duration, maxLoad int64, redirect bool) {
	mgr.Lock()
	defer mgr.Unlock()

	mgr.lines = append(mgr.lines, NewLine(addr, timeout, interval, maxLoad, redirect))
}

/* 开始检查所有 Line 状况 */
func (mgr *ProxyBase) StartCheckLines() {
	for i, line := range mgr.lines {
		line.Start(i)
	}
}

/* 停止检查所有 Line 状况 */
func (mgr *ProxyBase) StopCheckLines() {
	for _, line := range mgr.lines {
		line.Stop()
	}
}

// func (pbase *ProxyBase) GetBestLine() *Line {
// 	return pbase.GetBestLineWithoutLock()
// }
