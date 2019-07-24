package main

import (
	"fmt"
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"sync"
	"time"
)

var (
	CMD_ECHO = uint32(1)

	// tcp 反代
	tcpAddr = "localhost:10000"

	// websocket 反代
	websocketAddr = "ws://localhost:20000/gate/ws"
)

func onTcpEcho(client *net.TcpClient, msg net.IMessage) {
	log.Debug("tcp client onEcho from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
}

func onWebsocketEcho(client *net.WSClient, msg net.IMessage) {
	log.Debug("websocket client onEcho from %v: %v", client.Conn.RemoteAddr().String(), string(msg.Body()))
}

func main() {
	wg := sync.WaitGroup{}

	// tcp client
	{
		autoReconn := true
		netengine := net.NewTcpEngine()
		netengine.Handle(CMD_ECHO, onTcpEcho)
		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func(idx int) {
				defer wg.Done()

				client, err := net.NewTcpClient(tcpAddr, netengine, nil, autoReconn, nil)
				if err != nil {
					log.Panic("NewTcpClient failed: %v, %v", client, err)
				}

				for i := 0; true; i++ {
					err = client.SendMsg(net.NewMessage(CMD_ECHO, []byte(fmt.Sprintf("hello %v", i))))
					if err != nil {
						log.Error("tcp client echo failed: %v", err)
					}
					time.Sleep(time.Second)
				}
			}(i)
		}
	}

	// websocket client
	{
		for i := 0; i < 10; i++ {
			wg.Add(1)

			go func(idx int) {
				client, err := net.NewWebsocketClient(websocketAddr)
				if err != nil {
					log.Panic("NewWebsocketClient failed: %v, %v", err, time.Now())
				}

				// 初始化协议号
				client.Handle(CMD_ECHO, onWebsocketEcho)

				for i := 0; true; i++ {
					err = client.SendMsg(net.NewMessage(CMD_ECHO, []byte(fmt.Sprintf("hello %v", i))))
					if err != nil {
						log.Error("ws client echo failed: %v", err)
						break
					}
					time.Sleep(time.Second)
				}
			}(i)
		}
	}

	wg.Wait()
}
