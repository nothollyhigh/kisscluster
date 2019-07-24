package app

import (
	"fmt"
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"github.com/nothollyhigh/kiss/util"
	"kisscluster/proto"
	"sync"
	"time"
)

var (
	svrMgr = &SvrMgr{
		Plazas: map[string]*ServerInfo{},
		Games:  map[string]*ServerInfo{},

		updateInterval: time.Second * 5,
	}
)

type ServerInfo struct {
	proto.ServerInfo
	Client *net.TcpClient `json:"-"`
}

type SvrMgr struct {
	sync.RWMutex

	timer          *time.Timer
	updateInterval time.Duration

	Plazas map[string]*ServerInfo
	Games  map[string]*ServerInfo
}

func (mgr *SvrMgr) Add(svr *ServerInfo) (code int, err error) {
	log.Info("SvrMgr Add %v, %v", svr.Id, svr.Type)
	mgr.Lock()
	switch svr.Type {
	case proto.SERVER_TYPE_PLAZA:
		mgr.Plazas[svr.Id] = svr

		//同步游戏服务列表
		msg := proto.NewMessage(proto.CMD_CENTER_UPDATE_GAME_LIST_NOTIFY, &mgr.Games)
		svr.Client.SendMsg(msg)
	case proto.SERVER_TYPE_GAME:
		mgr.Games[svr.Id] = svr
	default:
		code = -1
		err = fmt.Errorf("invalid server type: '%v'", svr.Type)
		log.Error("SvrMgr Add failed, invalid server type: %v", svr.Type)
	}
	mgr.Unlock()

	if svr.Type == proto.SERVER_TYPE_GAME {
		mgr.UpdateServerList()
	}

	return
}

func (mgr *SvrMgr) Delete(svr *ServerInfo) {
	log.Info("SvrMgr Delete %v, %v", svr.Id, svr.Type)

	mgr.Lock()
	switch svr.Type {
	case proto.SERVER_TYPE_PLAZA:
		delete(mgr.Plazas, svr.Id)
	case proto.SERVER_TYPE_GAME:
		delete(mgr.Games, svr.Id)
	default:
		log.Error("SvrMgr Delete failed, invalid server type: %v", svr.Type)
	}
	mgr.Unlock()

	if svr.Type == proto.SERVER_TYPE_GAME {
		mgr.UpdateServerList()
	}
}

func (mgr *SvrMgr) UpdateServerList() {
	mgr.Lock()
	defer mgr.Unlock()

	msg := proto.NewMessage(proto.CMD_CENTER_UPDATE_GAME_LIST_NOTIFY, &mgr.Games)
	for _, plaza := range mgr.Plazas {
		plaza.Client.SendMsg(msg)
	}
	log.Info("UpdateServerList: %v", string(msg.Body()))
	mgr.timer.Reset(mgr.updateInterval)
}

func (mgr *SvrMgr) run() {
	if config.Refresh <= 0 {
		mgr.updateInterval = time.Second * 5
	} else {
		mgr.updateInterval = time.Second * time.Duration(config.Refresh)
	}
	mgr.timer = time.NewTimer(mgr.updateInterval)
	util.Go(func() {
		for {
			<-mgr.timer.C
			mgr.UpdateServerList()
		}
	})
}
