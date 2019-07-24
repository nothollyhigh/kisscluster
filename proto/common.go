package proto

var (
	SERVER_TYPE_PLAZA = "plaza"
	SERVER_TYPE_GAME  = "game"
)

type ServerInfo struct {
	Id   string
	Type string
	Info interface{}
}
