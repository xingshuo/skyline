package main

import (
	"context"
	"fmt"

	"github.com/xingshuo/skyline/defines"

	"github.com/xingshuo/skyline/utils"

	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/log"
)

func main() {
	c := make(chan struct{}, 1)
	skyline.Init("config.json", func(ctx context.Context) error {
		_, err := skyline.NewService("greetSender", &defines.DummyModule{}, 0)
		if err != nil {
			log.Fatalf("new gate svc failed: %v", err)
		}
		skyline.AsyncCallRemote(ctx, func(_ context.Context, peerName interface{}, err error) {
			utils.Assert(err == nil, fmt.Sprintf("do greet failed: %v", err))
			log.Infof("get reply from server %s", peerName)
			c <- struct{}{}
		}, "rpc_server", "greetReceiver", "Greeting", "lilei")
		return nil
	})

	<-c
	skyline.Exit()
}
