package defines

type SVC_HANDLE uint64

const (
	DEFAULT_MQ_SIZE = 1024
)

const (
	CtxKeySrcCluster = "SrcCluster"
	CtxKeySrcSvcName = "SrcSvcName"
	CtxKeyService    = "Service"
	CtxKeyRpcTimeout = "RpcTimeout" // 单位: time.Duration
)
