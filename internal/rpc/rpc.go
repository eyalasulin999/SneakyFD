package rpc

import (
	"sneakyfd/protobuf/messages"

	"google.golang.org/protobuf/proto"
)

type RPCHandler func([]byte) (uint32, proto.Message, error)

var (
	Handlers = map[uint32]RPCHandler{
		messages.MsgPingReq: PingRPC,
	}
)

func PingRPC(data []byte) (resType uint32, res proto.Message, err error) {
	pingReq := &messages.PingReq{}
	err = proto.Unmarshal(data, pingReq)
	if err != nil {
		return
	}
	resType = messages.MsgPingRes
	res = &messages.PingRes{Payload: pingReq.Payload}
	return
}
