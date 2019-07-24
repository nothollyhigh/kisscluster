package app

import (
	"crypto/tls"
	"encoding/binary"
	"github.com/gorilla/websocket"
	"github.com/nothollyhigh/kiss/log"
	knet "github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	DefaultSocketOpt = &knet.SocketOpt{
		NoDelay:           true,
		Keepalive:         false,
		ReadBufLen:        1024 * 8,
		WriteBufLen:       1024 * 8,
		ReadTimeout:       time.Second * 35,
		ReadHeaderTimeout: time.Second * 10,
		WriteTimeout:      time.Second * 5,
		MaxHeaderBytes:    4096,
	}

	DefaultUpgrader = &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

/* websocket 代理 */
type ProxyWebsocket struct {
	*ProxyBase
	Running       bool
	EnableTls     bool
	Listener      *net.TCPListener
	Heartbeat     time.Duration
	AliveTime     time.Duration
	RecvBlockTime time.Duration
	RecvBufLen    int
	SendBlockTime time.Duration
	SendBufLen    int
	Nodelay       bool
	ConnCount     uint64
	Certs         []XMLCert
	Routes        map[string]func(w http.ResponseWriter, r *http.Request)
}

func (pws *ProxyWebsocket) InitConn(conn *net.TCPConn) bool {
	if err := conn.SetKeepAlivePeriod(pws.AliveTime); err != nil {
		log.Info("ProxyWebsocket(TLS: %v) InitConn SetKeepAlivePeriod Err: %v", pws.EnableTls, err)
		return false
	}

	if err := conn.SetReadBuffer(pws.RecvBufLen); err != nil {
		log.Info("ProxyWebsocket(TLS: %v) InitConn SetReadBuffer Err: %v", pws.EnableTls, err)
		return false
	}
	if err := conn.SetWriteBuffer(pws.SendBufLen); err != nil {
		log.Info("ProxyWebsocket(TLS: %v) InitConn SetWriteBuffer Err: %v", pws.EnableTls, err)
		return false
	}
	if err := conn.SetNoDelay(pws.Nodelay); err != nil {
		log.Info("ProxyWebsocket(TLS: %v) InitConn SetNoDelay Err: %v", pws.EnableTls, err)
		return false
	}
	return true
}

func (pws *ProxyWebsocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := pws.Routes[r.URL.Path]; ok {
		h(w, r)
		return
	}
	http.NotFound(w, r)
}

func (pws *ProxyWebsocket) OnNew(w http.ResponseWriter, r *http.Request) {
	defer util.HandlePanic()

	atomic.AddUint64(&(pws.ConnCount), 1)

	var (
		serverConn *net.TCPConn
		tcpAddr    *net.TCPAddr
		wsaddr     = r.RemoteAddr
	)

	wsline := pws.GetBestLine()
	if wsline == nil {
		log.Info("Session(%s -> null, TLS: %v) Over, GetLineByAddr Failed", wsaddr, pws.EnableTls)
		http.NotFound(w, r)
		return
	}

	wsConn, err := DefaultUpgrader.Upgrade(w, r, nil)

	line := wsline

	connMgr.UpdateInNum(1)
	defer connMgr.UpdateInNum(-1)

	if tcpAddr, err = net.ResolveTCPAddr("tcp", line.Remote); err != nil {
		log.Info("Session(%s -> %s, TLS: %v) ResolveTCPAddr Err: %s", wsaddr, line.Remote, pws.EnableTls, err.Error())
		wsConn.Close()
		line.UpdateDelay(unreachableTime)
		line.UpdateFailedNum(1)
		connMgr.UpdateFailedNum(1)
		return
	}

	var (
		clientRecv int64 = 0
		clientSend int64 = 0
		serverRecv int64 = 0
		serverSend int64 = 0
	)

	s2c := func() {
		defer util.HandlePanic()
		defer func() {
			wsConn.Close()
			if serverConn != nil {
				serverConn.Close()
			}
		}()

		var headlen = 16
		var nread int
		var bodylen int
		var err error
		var buf = make([]byte, pws.RecvBufLen)
		for {
			serverConn.SetReadDeadline(time.Now().Add(pws.RecvBlockTime))
			nread, err = io.ReadFull(serverConn, buf[:headlen])
			if err != nil {
				wsConn.Close()
				log.Info("Session(%s -> %s, TLS: %v) Closed, Server Read Err: %s",
					wsaddr, line.Remote, pws.EnableTls, err.Error())
				break
			}

			serverRecv += int64(nread)
			connMgr.UpdateServerInSize(int64(nread))

			bodylen = int(binary.LittleEndian.Uint32(buf[:4]))
			if bodylen > 0 {
				if cap(buf) < headlen+bodylen {
					if (headlen+bodylen)%1024 != 0 {
						newBuf := make([]byte, 1024*((headlen+bodylen)/1024+1))
						copy(newBuf, buf)
						buf = newBuf
					} else {
						newBuf := make([]byte, headlen+bodylen)
						copy(newBuf, buf)
						buf = newBuf
					}

				}
				serverConn.SetReadDeadline(time.Now().Add(pws.RecvBlockTime))
				nread, err = io.ReadFull(serverConn, buf[headlen:headlen+bodylen])
				if err != nil {
					wsConn.Close()
					log.Info("Session(%s -> %s, TLS: %v) Closed, Server Read Err: %s",
						wsaddr, line.Remote, pws.EnableTls, err.Error())
					break
				}

				serverRecv += int64(nread)
				connMgr.UpdateServerInSize(int64(nread))

				nread += headlen
			}
			wsConn.SetWriteDeadline(time.Now().Add(pws.SendBlockTime))
			err = wsConn.WriteMessage(websocket.TextMessage, buf[:nread])
			if err != nil {
				log.Info("Session(%s -> %s, TLS: %v) Closed, Server WriteMessage Err: %s",
					wsaddr, line.Remote, pws.EnableTls, err.Error())
				break
			}

			serverSend += int64(nread)
			connMgr.UpdateServerOutSize(int64(nread))
		}
	}

	c2s := func() {
		defer func() {
			wsConn.Close()
			if serverConn != nil {
				serverConn.Close()
			}
		}()

		var nwrite int
		var err error
		var message []byte

		for {

			err = wsConn.SetReadDeadline(time.Now().Add(pws.RecvBlockTime))
			if err != nil {
				log.Info("Session(%s -> %s, TLS: %v) Closed, Client ReadMessage Err: %s",
					wsaddr, line.Remote, pws.EnableTls, err.Error())
				break
			}

			_, message, err = wsConn.ReadMessage()
			if err != nil {
				log.Info("Session(%s -> %s, TLS: %v) Closed, Client ReadMessage Err: %s",
					wsaddr, line.Remote, pws.EnableTls, err.Error())
				break
			}

			if serverConn == nil {
				t1 := time.Now()
				serverConn, err = net.DialTCP("tcp", nil, tcpAddr)
				if err != nil {
					log.Info("Session(%s -> %s, TLS: %v) DialTCP Err: %s",
						wsaddr, line.Remote, pws.EnableTls, err.Error())
					wsConn.Close()
					line.UpdateDelay(unreachableTime)
					line.UpdateFailedNum(1)
					connMgr.UpdateFailedNum(1)
					return
				}

				line.UpdateDelay(time.Since(t1))

				pws.InitConn(serverConn)

				line.UpdateLoad(1)
				defer line.UpdateLoad(-1)

				connMgr.UpdateOutNum(1)
				defer connMgr.UpdateOutNum(-1)

				connMgr.UpdateSuccessNum(1)

				log.Info("Session(%s -> %s, TLS: %v) Established", wsaddr, line.Remote, pws.EnableTls)

				if err = line.HandleRedirect(serverConn, wsaddr); err != nil {
					log.Info("Session(%s -> %s) HandleRedirect Failed: %s", wsaddr, line.Remote, err.Error())
					return
				}

				util.Go(s2c)
			}

			clientRecv += int64(len(message))
			connMgr.UpdateClientInSize(int64(len(message)))

			serverConn.SetWriteDeadline(time.Now().Add(pws.SendBlockTime))
			nwrite, err = serverConn.Write(message)
			if nwrite != len(message) || err != nil {
				log.Info("Session(%s -> %s, TLS: %v) Closed, Client Write Err: %s",
					wsaddr, line.Remote, pws.EnableTls, err.Error())
				break
			}

			clientSend += int64(nwrite)
			connMgr.UpdateClientOutSize(int64(nwrite))
		}
	}

	c2s()

	log.Info("Session(%s -> %s, TLS: %v) Over, DataInfo(CR: %d, CW: %d, SR: %d, SW: %d)",
		wsaddr, line.Remote, pws.EnableTls, clientRecv, clientSend, serverRecv, serverSend)
}

func (pws *ProxyWebsocket) Start() {
	if len(pws.lines) == 0 {
		log.Info("ProxyWebsocket(%v TLS: %v) Start Err: No Line !", pws.name, pws.EnableTls)
		return
	}

	util.Go(func() {
		pws.Lock()
		defer pws.Unlock()
		if !pws.Running {
			pws.Running = true

			util.Go(func() {
				l, err := knet.NewListener(pws.local, DefaultSocketOpt)
				if err != nil {
					log.Fatal("ProxyWebsocket(%v TLS: %v) NewListener Failed: %v", pws.name, pws.EnableTls, err)
				}

				s := &http.Server{
					Addr:              pws.local,
					Handler:           pws,
					ReadTimeout:       DefaultSocketOpt.ReadTimeout,
					ReadHeaderTimeout: DefaultSocketOpt.ReadHeaderTimeout,
					WriteTimeout:      DefaultSocketOpt.WriteTimeout,
					MaxHeaderBytes:    DefaultSocketOpt.MaxHeaderBytes,
				}

				if pws.EnableTls {
					if len(pws.Routes) == 0 {
						pws.Routes["/kissgate/wss"] = pws.OnNew
					}

					log.Info("ProxyWebsocket(%v TLS: %v) Running On: %s, Routes: %+v, Certs: %+v", pws.name, pws.EnableTls, pws.local, pws.Routes, pws.Certs)

					pws.StartCheckLines()
					defer pws.StopCheckLines()

					if len(pws.Certs) == 0 {
						log.Fatal("ProxyWebsocket(%v TLS: %v) ListenAndServeTLS Error: No Cert And Key Files", pws.name, pws.EnableTls)
					}

					s.TLSConfig = &tls.Config{}
					for _, v := range pws.Certs {
						cert, err := tls.LoadX509KeyPair(v.Certfile, v.Keyfile)
						if err != nil {
							log.Fatal("ProxyWebsocket(%v TLS: %v) tls.LoadX509KeyPair(%v, %v) Failed: %v", pws.name, pws.EnableTls, v.Certfile, v.Keyfile, err)
						}
						s.TLSConfig.Certificates = append(s.TLSConfig.Certificates, cert)
					}

					tlsListener := tls.NewListener(l, s.TLSConfig)

					if err := s.Serve(tlsListener); err != nil {
						log.Fatal("ProxyWebsocket(%v TLS: %v) Serve Error: %v", pws.name, pws.EnableTls, err)
					}
				} else {
					if len(pws.Routes) == 0 {
						pws.Routes["/kissgate/ws"] = pws.OnNew
					}

					log.Info("ProxyWebsocket(%v TLS: %v, Routes: %+v) Running On: %s", pws.name, pws.EnableTls, pws.Routes, pws.local)

					pws.StartCheckLines()
					defer pws.StopCheckLines()

					if err := s.Serve(l); err != nil {
						log.Fatal("ProxyWebsocket(TLS: %v) Serve Error: %v", pws.EnableTls, err)
					}
				}
			})
		}
	})
}

func (pws *ProxyWebsocket) Stop() {
	pws.Lock()
	defer pws.Unlock()
	if pws.Running {
		pws.Running = false
	}
}

func NewWebsocketProxy(name string, local string, paths []string, tls bool, certs []XMLCert) *ProxyWebsocket {
	pws := &ProxyWebsocket{
		Running:       false,
		EnableTls:     tls,
		Listener:      nil,
		Heartbeat:     DEFAULT_TCP_HEARTBEAT,
		AliveTime:     DEFAULT_TCP_KEEPALIVE_INTERVAL,
		RecvBlockTime: DEFAULT_TCP_READ_BLOCK_TIME,
		RecvBufLen:    DEFAULT_TCP_READ_BUF_LEN,
		SendBlockTime: DEFAULT_TCP_WRITE_BLOCK_TIME,
		SendBufLen:    DEFAULT_TCP_WRITE_BUF_LEN,
		Nodelay:       DEFAULT_TCP_NODELAY,
		Certs:         certs,
		Routes:        map[string]func(w http.ResponseWriter, r *http.Request){},

		ProxyBase: &ProxyBase{
			name:  name,
			ptype: PT_WEBSOCKET,
			local: local,
			lines: []*Line{},
		},
	}

	for _, path := range paths {
		pws.Routes[path] = pws.OnNew
	}

	return pws
}
