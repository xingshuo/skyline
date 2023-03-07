package defines

import "time"

type SVC_HANDLE uint64

const (
	DefaultMQSize       = 1024
	DefaultSSRpcTimeout = 6 * time.Second
)

const (
	CtxKeySrcCluster = "SrcCluster"
	CtxKeySrcSvcName = "SrcSvcName"
	CtxKeyService    = "Service"
	CtxKeyRpcTimeout = "RpcTimeout" // 单位: time.Duration
)
