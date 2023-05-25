package defines

import (
	"errors"
)

var (
	ErrOK              = errors.New("OK")
	ErrRpcTimeout      = errors.New("RpcTimeout")
	ErrTransportClosed = errors.New("TransportClosed")
	ErrSvcNoExist      = errors.New("ServiceNoExist")
	ErrAddrListenAgain = errors.New("AddrListenAgain")
)
