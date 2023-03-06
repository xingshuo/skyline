package proto

import (
	"fmt"

	"github.com/xingshuo/skyline/defines"
)

const (
	PTYPE_TIMER       = 1 // service timer
	PTYPE_REQUEST     = 2 // service rpc request
	PTYPE_RESPONSE    = 3 // service rpc response
	PTYPE_CLUSTER_REQ = 4
	PTYPE_CLUSTER_RSP = 5
	PTYPE_SPAWN       = 6
	PTYPE_ASYNC_CB    = 7
)

type MsgType int

func (mt MsgType) String() string {
	switch mt {
	case PTYPE_TIMER:
		return "TIMER"
	case PTYPE_REQUEST:
		return "REQUEST"
	case PTYPE_RESPONSE:
		return "RESPONSE"
	case PTYPE_CLUSTER_REQ:
		return "CLUSTER_REQ"
	case PTYPE_CLUSTER_RSP:
		return "CLUSTER_RSP"
	case PTYPE_SPAWN:
		return "SPAWN"
	case PTYPE_ASYNC_CB:
		return "ASYNC_CB"
	default:
		return "unknown"
	}
}

type Message struct {
	Source  defines.SVC_HANDLE
	MsgType MsgType
	Session uint32
	Data    []interface{}
}

func (m *Message) String() string {
	return fmt.Sprintf("msg:[%d,%s,%d]", m.Source, m.MsgType, m.Session)
}

type RpcResponse struct {
	Reply interface{}
	Err   error
}

type ClusterRequest struct {
	SrcCluster string
	SrcService string
	Request    []byte
}

type ClusterResponse struct {
	SrcCluster string
	SrcService string
	Response   []byte
	ErrMsg     string
}
