package app

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/proto"
	"time"
)

var (
	centerSession *net.RpcClient

	gameList = map[string]*proto.ServerInfo{}
)

func updatePlazaInfo() {
	var (
		req = &proto.CenterUpdateServerInfoReq{
			proto.ServerInfo{
				Id:   config.SvrID,
				Type: proto.SERVER_TYPE_PLAZA,
			},
		}
		rsp = &proto.CenterUpdateServerInfoRsp{}
	)

	err := centerSession.Call(proto.RPC_METHOD_UPDATE_SERVER_INFO, req, rsp, time.Second*3)
	if err != nil {
		log.Error("onConnectedCenter updateInfo failed: %v", err)
		return
	}

	if rsp.Code == 0 {
		log.Info("onConnectedCenter updateInfo success")
	} else {
		log.Error("onConnectedCenter updateInfo failed, code: %v, msg: %v", rsp.Code, rsp.Msg)
	}
}

func onConnectedCenter(client *net.RpcClient) {
	updatePlazaInfo()
}

func onUpdateGameListNotify(client *net.TcpClient, msg net.IMessage) {
	var (
		serverList = map[string]*proto.ServerInfo{}
	)

	err := proto.Unmarshal(msg.Body(), &serverList)
	if err != nil {
		log.Error("onUpdateGameListNotify bind failed: %v", err)
	} else {
		log.Info("onUpdateGameListNotify success: %v", string(msg.Body()))
		gameList = serverList
		userMgr.BroadcastGameList()
	}
}

func startCenterSession() {
	var (
		err       error
		netengine = net.NewTcpEngine()
	)
	netengine.Handle(proto.CMD_CENTER_UPDATE_GAME_LIST_NOTIFY, onUpdateGameListNotify)

	centerSession, err = net.NewRpcClient(config.CenterAddr, netengine, nil, onConnectedCenter)
	if err != nil {
		log.Error("NewRpcClient failed: %v", err)
		panic(err)
	}

	updatePlazaInfo()
}

func stopCenterSession() {
	util.Go(func() {
		centerSession.Shutdown()
	})
}
