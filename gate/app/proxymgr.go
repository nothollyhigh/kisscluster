package app

import (
	"fmt"
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"strings"
	"time"
)

var (
	proxyMgr = &ProxyMgr{Proxys: make(map[string]IProxy)}
)

type ProxyMgr struct {
	Proxys map[string]IProxy
}

func (mgr *ProxyMgr) AddProxy(name string, proxy IProxy) {
	if _, ok := mgr.Proxys[name]; ok {
		log.Fatal("Duplicate Proxy Name: %v", name)
	}

	mgr.Proxys[name] = proxy
}

func (mgr *ProxyMgr) InitPorxy() {
	options := xmlconfig.Options
	proxy := xmlconfig.Proxy

	DEFAULT_TCP_REDIRECT = options.Redirect

	DEFAULT_TCP_CHECKLINE_INTERVAL = time.Second * time.Duration(options.Heartbeat.Interval)
	DEFAULT_TCP_CHECKLINE_TIMEOUT = time.Second * time.Duration(options.Heartbeat.Timeout)

	addrs, err := net.GetLocalAddr()
	if err != nil {
		log.Fatal("Init GetLocalAddr() Err: ", err)
	}
	// for i, addr := range addrs {
	// 	log.Info("Local Addr Info, Addr[%d]: %s", i, addr)
	// }

	isLineValid := func(port string, lineaddr string) bool {
		for _, localaddr := range addrs {
			if fmt.Sprintf("%s:%s", localaddr, port) == lineaddr {
				return false
			}
		}
		return true
	}

	/* 创建并启动一个TcpProxy */
	newOneTcpProxy := func(name string, addr string, redictip bool, nodes []XMLNode) {
		ptcp := NewTcpProxy(name, addr)
		port := strings.Split(addr, ":")[0]
		for _, node := range nodes {
			if !isLineValid(port, node.Addr) {
				log.Fatal("Proxy(%s, %s) AddLine Error: Recursive, Shouldn't Use The Proxy Self's Addr(%s) As Target Addr", name, node.Addr, node.Addr)
			}
			ptcp.AddLine(node.Addr, DEFAULT_TCP_CHECKLINE_TIMEOUT, DEFAULT_TCP_CHECKLINE_INTERVAL, node.Maxload, redictip)
		}
		ptcp.Start()
		mgr.AddProxy(name, ptcp)
	}

	/* 创建一个WebsocketProxy */
	newOneWSProxy := func(name string, addr string, redictip bool, routes []XMLRoute, nodes []XMLNode, tls bool, certs []XMLCert) {
		paths := []string{}
		for _, route := range routes {
			paths = append(paths, route.Path)
		}
		pws := NewWebsocketProxy(name, addr, paths, tls, certs)
		port := strings.Split(addr, ":")[0]

		for _, node := range nodes {
			if !isLineValid(port, node.Addr) {
				log.Fatal("Proxy(%s, %s) AddLine Error: Recursive, Shouldn't Use The Proxy Self's Addr(%s) As Target Addr", name, node.Addr, node.Addr)
			}
			pws.AddLine(node.Addr, DEFAULT_TCP_CHECKLINE_TIMEOUT, DEFAULT_TCP_CHECKLINE_INTERVAL, node.Maxload, redictip)
		}
		pws.Start()
		mgr.AddProxy(name, pws)
	}

	for _, line := range proxy.Lines {
		switch line.Type {
		case PT_TCP:
			if line.Redirect == "" {
				newOneTcpProxy(line.Name, line.Addr, DEFAULT_TCP_REDIRECT, line.Nodes)
			} else {
				newOneTcpProxy(line.Name, line.Addr, line.Redirect == "true", line.Nodes)
			}

		case PT_WEBSOCKET:
			if line.Redirect == "" {
				newOneWSProxy(line.Name, line.Addr, DEFAULT_TCP_REDIRECT, line.Routes, line.Nodes, line.TLS, line.Certs)
			} else {
				newOneWSProxy(line.Name, line.Addr, line.Redirect == "true", line.Routes, line.Nodes, line.TLS, line.Certs)
			}
		}
	}
}
