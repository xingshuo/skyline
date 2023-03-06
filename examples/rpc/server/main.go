package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/xingshuo/skyline/skeleton"

	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/log"
)

type RpcServer struct {
	name string
}

func (s *RpcServer) OnInit(ctx context.Context) error {
	return nil
}

func (s *RpcServer) OnExit() {

}

func (s *RpcServer) Greeting(ctx context.Context, name string) (interface{}, error) {
	log.Infof("recv greeting from %s", name)
	return s.name, nil
}

func main() {
	skyline.Init("config.json")
	_, err := skyline.NewService("greetReceiver", skeleton.NewModule(&RpcServer{"richard"}, nil), 0)
	if err != nil {
		log.Fatalf("new gate svc failed: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	<-c

	skyline.Exit()
}
