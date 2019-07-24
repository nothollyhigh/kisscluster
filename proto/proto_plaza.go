package proto

const (
	CMD_PLAZA_LOGIN_REQ        uint32 = 1001 // 登录请求
	CMD_PLAZA_LOGIN_RSP        uint32 = 1002 // 登录响应
	CMD_PLAZA_GAME_LIST_NOTIFY uint32 = 1003 // 游戏服务列表通知
)

type PlazaLoginReq struct {
}

type PlazaLoginRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Name string `json:"name"`
}

type BroadcastNotify struct {
	Msg string `json:"msg"`
}
