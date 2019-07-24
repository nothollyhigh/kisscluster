package app

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/util"
	"net"
	"time"
)

/* tcp 代理 */
type ProxyTcp struct {
	*ProxyBase
	Running       bool
	Listener      *net.TCPListener
	Heartbeat     time.Duration
	AliveTime     time.Duration
	RecvBlockTime time.Duration
	RecvBufLen    int
	SendBlockTime time.Duration
	SendBufLen    int
	Nodelay       bool

	ConnCount uint64
}

func (ptcp *ProxyTcp) InitConn(conn *net.TCPConn) bool {
	if err := conn.SetKeepAlivePeriod(ptcp.AliveTime); err != nil {
		log.Info("ProxyTcp InitConn SetKeepAlivePeriod Err: %v", err)
		return false
	}

	if err := conn.SetReadBuffer(ptcp.RecvBufLen); err != nil {
		log.Info("ProxyTcp InitConn SetReadBuffer Err: %v", err)
		return false
	}
	if err := conn.SetWriteBuffer(ptcp.SendBufLen); err != nil {
		log.Info("ProxyTcp InitConn SetWriteBuffer Err: %v", err)
		return false
	}
	if err := conn.SetNoDelay(ptcp.Nodelay); err != nil {
		log.Info("ProxyTcp InitConn SetNoDelay Err: %v", err)
		return false
	}
	return true
}

func (ptcp *ProxyTcp) OnNew(clientConn *net.TCPConn) {
	defer util.HandlePanic()

	var (
		line       *Line
		serverConn *net.TCPConn
		tcpAddr    *net.TCPAddr
		clientAddr = clientConn.RemoteAddr().String()
	)

	connMgr.UpdateInNum(1)
	defer connMgr.UpdateInNum(-1)

	line = ptcp.GetBestLine()
	if line == nil {
		log.Info("Session(%s -> null) Failed, GetBestLine Failed", clientAddr)
		clientConn.Close()
		return
	}

	var (
		clientRecv int64 = 0
		clientSend int64 = 0
		serverRecv int64 = 0
		serverSend int64 = 0
	)

	ptcp.InitConn(clientConn)

	s2cCor := func() {
		defer util.HandlePanic()
		defer func() {
			clientConn.Close()
			if serverConn != nil {
				serverConn.Close()
			}
		}()
		var nread int
		var nwrite int
		var err error
		var buf = make([]byte, ptcp.RecvBufLen)
		for {
			if err = serverConn.SetReadDeadline(time.Now().Add(ptcp.RecvBlockTime)); err != nil {
				log.Info("Session(%s -> %s) Closed, Server SetReadDeadline Err: %s", clientAddr, line.Remote, err.Error())
				break
			}
			nread, err = serverConn.Read(buf)
			if err != nil {
				log.Info("Session(%s -> %s) Closed, Server Read Err: %s", clientAddr, line.Remote, err.Error())
				break
			}

			serverRecv += int64(nread)
			connMgr.UpdateServerInSize(int64(nread))

			if err = clientConn.SetWriteDeadline(time.Now().Add(ptcp.SendBlockTime)); err != nil {
				log.Info("Session(%s -> %s) Closed, Server SetWriteDeadline Err: %s", clientAddr, line.Remote, err.Error())
				break
			}
			nwrite, err = clientConn.Write(buf[:nread])
			if nwrite != nread || err != nil {
				log.Info("Session(%s -> %s) Closed, Server Write Err: %s", clientAddr, line.Remote, err.Error())
				break
			}

			serverSend += int64(nwrite)
			connMgr.UpdateServerOutSize(int64(nwrite))
		}
	}

	c2sCor := func() {
		defer func() {
			clientConn.Close()
			if serverConn != nil {
				serverConn.Close()
			}
		}()
		var nread int
		var nwrite int
		var err error
		var buf = make([]byte, ptcp.RecvBufLen)

		for {
			if err = clientConn.SetReadDeadline(time.Now().Add(ptcp.RecvBlockTime)); err != nil {
				log.Info("Session(%s -> %s) Closed, Client SetReadDeadline Err: %s", clientAddr, line.Remote, err.Error())
				break
			}
			nread, err = clientConn.Read(buf)
			if err != nil {
				log.Info("Session(%s -> %s) Closed, Client Read Err: %s", clientAddr, line.Remote, err.Error())
				break
			}

			if serverConn == nil {
				if tcpAddr, err = net.ResolveTCPAddr("tcp", line.Remote); err != nil {
					log.Info("Session(%s -> %s) ResolveTCPAddr Failed, Client SetReadDeadline Err: %s", clientAddr, line.Remote, err.Error())
					clientConn.Close()
					line.UpdateDelay(unreachableTime)
					line.UpdateFailedNum(1)
					connMgr.UpdateFailedNum(1)
					return
				}

				t1 := time.Now()
				serverConn, err = net.DialTCP("tcp", nil, tcpAddr)
				if err != nil {
					log.Info("Session(%s -> %s) DailTCP Faield", clientAddr, line.Remote)
					clientConn.Close()
					line.UpdateDelay(unreachableTime)
					line.UpdateFailedNum(1)
					connMgr.UpdateFailedNum(1)
					return
				}

				line.UpdateDelay(time.Since(t1))

				line.UpdateLoad(1)
				defer line.UpdateLoad(-1)

				connMgr.UpdateOutNum(1)
				defer connMgr.UpdateOutNum(-1)

				connMgr.UpdateSuccessNum(1)

				log.Info("Session(%s -> %s) Established", clientAddr, line.Remote)

				ptcp.InitConn(serverConn)

				if err = line.HandleRedirect(serverConn, clientAddr); err != nil {
					log.Info("Session(%s -> %s) HandleRedirect Failed: %s", clientAddr, line.Remote, err.Error())
					return
				}

				util.Go(s2cCor)
			}

			clientRecv += int64(nread)
			connMgr.UpdateClientInSize(int64(nread))

			if err = serverConn.SetWriteDeadline(time.Now().Add(ptcp.SendBlockTime)); err != nil {
				log.Info("Session(%s -> %s) Closed, Client SetWriteDeadline Err: %s", clientAddr, line.Remote, err.Error())
				break
			}
			nwrite, err = serverConn.Write(buf[:nread])
			if nwrite != nread || err != nil {
				log.Info("Session(%s -> %s) Closed, Client Write Err: %s", clientAddr, line.Remote, err.Error())
				break
			}
			clientSend += int64(nwrite)
			connMgr.UpdateClientOutSize(int64(nwrite))
		}
	}

	c2sCor()

	log.Info("Session(%s -> %s) Over, DataInfo(CR: %d, CW: %d, SR: %d, SW: %d)",
		clientAddr, line.Remote, clientRecv, clientSend, serverRecv, serverSend)

	return
}

func (ptcp *ProxyTcp) start() {
	ptcp.Lock()
	defer ptcp.Unlock()
	if ptcp.Running {
		return
	}
	var (
		err     error
		conn    *net.TCPConn
		tcpAddr *net.TCPAddr
	)

	tcpAddr, err = net.ResolveTCPAddr("tcp4", ptcp.local)
	if err != nil {
		log.Error("ProxyTcp(%v) ListenAndServe() ResolveTCPAddr Err: %v", ptcp.name, err)
		return
	}

	ptcp.Listener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal("ProxyTcp(%v) ListenAndServe() ListenTCP Err: %v", ptcp.name, err)
		return
	}

	ptcp.Running = true
	ptcp.ConnCount = 0

	log.Info("ProxyTcp(%v) Running On: %s", ptcp.name, ptcp.local)

	ptcp.StartCheckLines()

	util.Go(func() {
		defer ptcp.StopCheckLines()

		for {
			if !ptcp.Running {
				break
			}

			conn, err = ptcp.Listener.AcceptTCP()

			if err != nil {
				log.Info("AcceptTCP Err: %s", err.Error())
			} else {
				ptcp.ConnCount++

				log.Info("OnNewConn: (%s <- %s)", ptcp.local, conn.RemoteAddr().String())

				util.Go(func() {
					ptcp.OnNew(conn)
				})
			}
		}
	})
}

func (ptcp *ProxyTcp) Start() {
	if len(ptcp.lines) == 0 {
		log.Error("ProxyTcp(%v) Start Err: No Line !", ptcp.name)
		return
	}
	util.Go(ptcp.start)
}

func (ptcp *ProxyTcp) Stop() {
	ptcp.Lock()
	defer ptcp.Unlock()
	if ptcp.Running {
		ptcp.Running = false
		ptcp.Listener.Close()
	}
}

func NewTcpProxy(name string, local string) *ProxyTcp {
	ptcp := &ProxyTcp{
		Running:       false,
		Listener:      nil,
		Heartbeat:     DEFAULT_TCP_HEARTBEAT,
		AliveTime:     DEFAULT_TCP_KEEPALIVE_INTERVAL,
		RecvBlockTime: DEFAULT_TCP_READ_BLOCK_TIME,
		RecvBufLen:    DEFAULT_TCP_READ_BUF_LEN,
		SendBlockTime: DEFAULT_TCP_WRITE_BLOCK_TIME,
		SendBufLen:    DEFAULT_TCP_WRITE_BUF_LEN,
		Nodelay:       DEFAULT_TCP_NODELAY,

		ProxyBase: &ProxyBase{
			name:  name,
			ptype: PT_TCP,
			local: local,
			lines: []*Line{},
		},
	}

	return ptcp
}
