package skynet_cluster

import (
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
)

var (
	client *Client
	server *Server
)

func Init(confPath string) {
	if client != nil || server != nil {
		log.Fatal("skynet plugin already init")
	}
	client = &Client{}
	err := client.Init(confPath)
	if err != nil {
		log.Fatalf("skynet plugin init failed:%v", err)
	}
	server = &Server{
		gates: make(map[string]*netframe.Listener),
	}
	log.Info("skynet plugin init done")
}

func Exit() {
	if client != nil {
		client.Exit()
	}
	if server != nil {
		server.Exit()
	}
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
