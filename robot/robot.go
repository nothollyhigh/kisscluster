package main

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/proto"
	"os"
	"syscall"
	"time"
)

var (
	plazaAddr = "ws://localhost:11000/gate/ws"
)

type Robot struct {
	Client *net.WSClient
}

func (robot *Robot) onPlazaLoginRsp(cli *net.WSClient, msg net.IMessage) {
	var (
		rsp = &proto.PlazaLoginRsp{}
	)

	err := proto.Unmarshal(msg.Body(), rsp)
	if err != nil {
		log.Error("onPlazaLoginRsp Unmarshal failed: %v", err)
		return
	}

	if rsp.Code != 0 {
		log.Error("onPlazaLoginRsp failed: %v, %v", rsp.Code, rsp.Msg)
		return
	}

	log.Info("onPlazaLoginRsp success, name: '%v'", rsp.Name)
}

func (robot *Robot) onGameList(cli *net.WSClient, msg net.IMessage) {
	log.Info("onGameList: %v", string(msg.Body()))
}

func NewRobot(addr string) (*Robot, error) {
	cli, err := net.NewWebsocketClient(addr)
	if err != nil {
		log.Error("NewWebsocketTLSClient failed: %v, %v", err, time.Now())
		return nil, err
	}

	robot := &Robot{
		Client: cli,
	}

	cli.Handle(proto.CMD_PLAZA_LOGIN_RSP, robot.onPlazaLoginRsp)
	cli.Handle(proto.CMD_PLAZA_GAME_LIST_NOTIFY, robot.onGameList)

	// 登录
	msg := proto.NewMessage(proto.CMD_PLAZA_LOGIN_REQ, &proto.PlazaLoginReq{})
	cli.SendMsg(msg)

	// 心跳
	util.Go(func() {
		cli.Keepalive(time.Second * 5)
	})

	return robot, nil
}

func main() {
	NewRobot(plazaAddr)

	util.HandleSignal(func(sig os.Signal) {
		if sig == syscall.SIGTERM || sig == syscall.SIGINT {
			os.Exit(0)
		}
	})
}
