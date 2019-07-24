package proto

import (
	"github.com/json-iterator/go"
	"github.com/nothollyhigh/kiss/net"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

var (
	Empty *struct{} = nil

	DefaultCodec ICodec = json
)

type ICodec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error

	MarshalToString(v interface{}) (string, error)
	UnmarshalFromString(s string, v interface{}) error
}

func SetCodec(c ICodec) {
	DefaultCodec = c
}

func Marshal(v interface{}) ([]byte, error) {
	return DefaultCodec.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return DefaultCodec.Unmarshal(data, v)
}

func MarshalToString(v interface{}) (string, error) {
	return DefaultCodec.MarshalToString(v)
}

func UnmarshalFromString(s string, v interface{}) error {
	return DefaultCodec.UnmarshalFromString(s, v)
}

func NewMessage(cmd uint32, v interface{}) *net.Message {
	data, ok := v.([]byte)
	if !ok {
		data, _ = Marshal(v)
	}
	return net.NewMessage(cmd, data)
}
