package proto

const (
	RPC_METHOD_UPDATE_SERVER_INFO = "update server info"

	CMD_CENTER_UPDATE_GAME_LIST_NOTIFY uint32 = 1
)

type CenterUpdateServerInfoReq struct {
	ServerInfo
}

type CenterUpdateServerInfoRsp struct {
	Code int
	Msg  string
}
