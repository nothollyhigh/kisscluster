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
)

func updateGameInfo() {
	var (
		req = &proto.CenterUpdateServerInfoReq{
			proto.ServerInfo{
				Id:   config.SvrID,
				Type: proto.SERVER_TYPE_GAME,
				Info: map[string]interface{}{
					"addr": config.SvrAddr,
				},
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
	centerSession = client
	updateGameInfo()
}

func startCenterSession() {
	var (
		err       error
		netengine = net.NewTcpEngine()
	)

	// netengine.Handle(proto.CMD_CENTER_UPDATE_GAME_LIST_NOTIFY, onUpdateGameListNotify)

	centerSession, err = net.NewRpcClient(config.CenterAddr, netengine, nil, onConnectedCenter)
	if err != nil {
		log.Error("NewTcpClient failed: %v", err)
		panic(err)
	}

	updateGameInfo()
}

func stopCenterSession() {
	util.Go(func() {
		centerSession.Shutdown()
	})
}
