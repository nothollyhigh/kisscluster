package app

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/proto"
)

var (
	server = net.NewTcpServer("Center")
)

func onUpdateServerInfo(ctx *net.RpcContext) {
	var (
		err  error
		code int
		req  = &proto.CenterUpdateServerInfoReq{}
		rsp  = &proto.CenterUpdateServerInfoRsp{}
	)

	if err = ctx.Bind(req); err != nil {
		rsp.Code = -1
		rsp.Msg = "invalid body"
		ctx.Write(rsp)
		return
	}

	svr := &ServerInfo{req.ServerInfo, ctx.Client()}
	code, err = svrMgr.Add(svr)
	if err != nil {
		rsp.Code = code
		rsp.Msg = err.Error()
	} else {
		ctx.Client().OnClose("DeleServer", func(c *net.TcpClient) {
			svrMgr.Delete(svr)
		})
	}

	ctx.Write(rsp)

	log.Info("onUpdateServerInfo: %v", string(ctx.Body()))
}

func startServer() {
	server.HandleRpcMethod(proto.RPC_METHOD_UPDATE_SERVER_INFO, onUpdateServerInfo)

	util.Go(func() {
		server.Start(config.SvrAddr)
	})
}
