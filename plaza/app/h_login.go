package app

import (
	"fmt"
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"kisscluster/proto"
	"sync/atomic"
)

var (
	count int64 = 0
)

func onPlazaLoginReq(client *net.TcpClient, msg net.IMessage) {
	var (
		err error
		req = &proto.PlazaLoginReq{}
		rsp = &proto.PlazaLoginRsp{}
	)

	if err = json.Unmarshal(msg.Body(), req); err != nil {
		rsp.Code = -1
		rsp.Msg = "invaid json"
		client.SendMsgWithCallback(proto.NewMessage(proto.CMD_PLAZA_LOGIN_RSP, rsp), userMgr.KickClient)
		return
	}

	rsp.Msg = "登录成功"
	rsp.Name = fmt.Sprintf("guest_%v", atomic.AddInt64(&count, 1))

	userMgr.Add(rsp.Name, client)
	client.OnClose("disconnected", func(*net.TcpClient) {
		userMgr.Delete(rsp.Name)
	})

	client.SendMsg(proto.NewMessage(proto.CMD_PLAZA_LOGIN_RSP, rsp))

	client.SendMsg(proto.NewMessage(proto.CMD_PLAZA_GAME_LIST_NOTIFY, gameList))

	log.Info("onPlazaLoginReq success: %v", rsp.Name)
}
