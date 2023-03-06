package main

import (
	"context"
	"fmt"

	"github.com/xingshuo/skyline/utils"

	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/log"
)

var rpcCliModule = &rpcClient{"lilei"}

type rpcClient struct {
	name string
}

func (c *rpcClient) Init(ctx context.Context) error {
	return nil
}

func (c *rpcClient) LocalProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (c *rpcClient) RemoteProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (c *rpcClient) Exit() {

}

func main() {
	skyline.Init("config.json")
	cliSvc, err := skyline.NewService("greetSender", rpcCliModule, 0)
	if err != nil {
		log.Fatalf("new gate svc failed: %v", err)
	}
	c := make(chan struct{}, 1)
	cliSvc.AsyncCallRemote(context.Background(), func(ctx context.Context, peerName interface{}, err error) {
		utils.Assert(err == nil, fmt.Sprintf("do greet failed: %v", err))
		log.Infof("get reply from server %s", peerName)
		c <- struct{}{}
	}, "rpc_server", "greetReceiver", "Greeting", rpcCliModule.name)

	<-c
	skyline.Exit()
}
