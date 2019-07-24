package app

import (
	"github.com/nothollyhigh/kiss/log"
	"github.com/nothollyhigh/kiss/net"
	"kisscluster/proto"
	"sync"
)

var (
	userMgr = &UserMgr{
		users: map[string]*net.TcpClient{},
	}
)

type UserMgr struct {
	sync.RWMutex
	users map[string]*net.TcpClient
}

func (mgr *UserMgr) Add(name string, client *net.TcpClient) {
	mgr.Lock()
	defer mgr.Unlock()

	mgr.users[name] = client
}

func (mgr *UserMgr) Delete(name string) {
	mgr.Lock()
	defer mgr.Unlock()

	delete(mgr.users, name)
}

func (mgr *UserMgr) KickClient(client *net.TcpClient, err error) {
	client.Stop()
}

func (mgr *UserMgr) BroadcastGameList() {
	mgr.RLock()
	defer mgr.RUnlock()

	msg := proto.NewMessage(proto.CMD_PLAZA_GAME_LIST_NOTIFY, gameList)

	for _, client := range mgr.users {
		client.SendMsg(msg)
	}

	log.Info("BroadcastGameList to %d clients: %v", len(mgr.users), string(msg.Body()))
}

// func (mgr *UserMgr) BroadcastGameListLoop() {
// 	for i := 0; true; i++ {
// 		time.Sleep(time.Second * 5)
// 		mgr.BroadcastGameList()
// 	}
// }

// func startUpdateServerListTask() {
// 	util.Go(userMgr.BroadcastGameListLoop)
// }
