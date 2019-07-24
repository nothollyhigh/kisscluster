package app

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/proto"
	"os"
	"time"
)

var (
	tcpServer = net.NewTcpServer("Plaza")
)

func startTcpServer() {
	tcpServer.Handle(proto.CMD_PLAZA_LOGIN_REQ, onPlazaLoginReq)

	util.Go(func() {
		tcpServer.Start(config.SvrAddr)
	})
}

func stopTcpServer() {
	tcpServer.StopWithTimeout(time.Second*5, func() {
		log.Error("Plaza Stop timeout")
		os.Exit(-1)
	})
}
