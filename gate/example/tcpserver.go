package main

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"time"
)

var ()

const (
	CMD_ECHO = uint32(1)
)

func onEcho(client *net.TcpClient, msg net.IMessage) {
	log.Info("tcp server %v onEcho: %v", client.Conn.LocalAddr().String(), string(msg.Body()))
	client.SendMsg(msg)
}

func serve(addr string) {
	server := net.NewTcpServer("Echo")

	// 初始化协议号
	server.Handle(CMD_ECHO, onEcho)

	server.Serve(addr, time.Second*5)
}

func main() {
	// 网关反代websocket的后端线路
	go serve(":10001")
	go serve(":10002")

	// 网关反代tcp的后端线路
	go serve(":20001")
	serve(":20002")
}
