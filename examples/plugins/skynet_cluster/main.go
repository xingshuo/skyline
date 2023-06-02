package main

import (
	"context"
	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/plugins/skynet_cluster"
	"github.com/xingshuo/skyline/seri"
	"github.com/xingshuo/skyline/skeleton"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	luaType = 10
)

type RouterAgent struct {
}

func (ra *RouterAgent) OnInit(ctx context.Context) error {
	return nil
}

func (ra *RouterAgent) OnExit() {

}

func (ra *RouterAgent) Request(ctx context.Context, peerName string) (interface{}, error) {
	ts := time.Now().Unix()
	log.Infof("recv request msg from [%s], ts: %v", peerName, ts)
	m := &seri.Table{
		Array: []interface{}{
			-65538, "a", false, 3.1415926,
		},
		Hashmap: map[interface{}]interface{}{
			"user":   "Bob",
			"passwd": "dzqm0701",
			"age":    33,
		},
	}
	skynet_cluster.Send(peerName, ".routersender", luaType, "Response", ts, m)
	return nil, nil
}

func main() {
	skyline.Init("config.json", func(ctx context.Context) error {
		_, err := skyline.NewService("routeragent", skeleton.NewModule(&RouterAgent{}, nil), 0)
		if err != nil {
			log.Fatalf("new routeragent svc failed: %v", err)
		}
		return nil
	})
	skynet_cluster.Init("skynet_clustername.json")
	err := skynet_cluster.Open("go_server")
	if err != nil {
		log.Fatalf("skynet_cluster[Open] failed: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	<-c

	skynet_cluster.Exit()
	skyline.Exit()
}
