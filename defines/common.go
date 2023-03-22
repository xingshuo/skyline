package defines

import (
	"context"
	"time"
)

type SVC_HANDLE uint64

const (
	DefaultMQSize       = 1024
	DefaultSSRpcTimeout = 6 * time.Second
	BootstrapSvcName    = "bootstrap"
)

const (
	CtxKeySrcCluster = "SrcCluster"
	CtxKeySrcSvcName = "SrcSvcName"
	CtxKeyService    = "Service"
	CtxKeyRpcTimeout = "RpcTimeout" // 单位: time.Duration
)

type DummyModule struct {
}

func (d *DummyModule) Init(ctx context.Context) error {
	return nil
}

func (d *DummyModule) LocalProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (d *DummyModule) RemoteProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (d *DummyModule) Exit() {

}
