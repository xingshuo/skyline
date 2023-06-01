package skynet_cluster

import (
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
)

var (
	client *skynetClient
	server *skynetServer
)

func Init(confPath string) {
	if client != nil || server != nil {
		log.Fatal("skynet-cluster plugin already init")
	}
	client = &skynetClient{}
	err := client.Init(confPath)
	if err != nil {
		log.Fatalf("skynet-cluster plugin init failed:%v", err)
	}
	server = &skynetServer{
		gates: make(map[string]*netframe.Listener),
	}
	log.Info("skynet-cluster plugin init done")
}

func Exit() {
	if client != nil {
		client.Exit()
	}
	if server != nil {
		server.Exit()
	}
	log.Info("skynet-cluster plugin exit")
}

// equal to `cluster.open`
func Open(clusterName string) error {
	return server.Listen(clusterName)
}

// equal to `cluster.send`
func Send(clusterName, svcName string, protoType int, args ...interface{}) error {
	return client.Send(clusterName, svcName, protoType, args...)
}

// equal to `cluster.reload`
func Reload() error {
	return client.Reload()
}
