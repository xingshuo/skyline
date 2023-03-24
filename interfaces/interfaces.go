package interfaces

import (
	"context"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/proto"
)

type App interface {
	PushToService(svcName string, source defines.SVC_HANDLE, msgType proto.MsgType, session uint32, data ...interface{}) error
}

type Module interface {
	Init(ctx context.Context) error
	LocalProcess(ctx context.Context, args ...interface{}) (interface{}, error)
	RemoteProcess(ctx context.Context, args ...interface{}) (interface{}, error)
	Exit()
}

type Codec interface {
	EncodeRequest(args ...interface{}) ([]byte, error)
	DecodeRequest(data []byte) ([]interface{}, error)

	EncodeResponse(reply interface{}) ([]byte, error)
	DecodeResponse(data []byte) (interface{}, error)
}
