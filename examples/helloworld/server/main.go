package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/xingshuo/skyline/skeleton"

	"github.com/xingshuo/skyline/netframe"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline"
)

type listReceiver struct {
}

func (r *listReceiver) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 2, true
}

func (r *listReceiver) OnConnected(c netframe.Conn) error {
	skyline.Send(context.Background(), gateSvcName, "NewConn", c)
	return nil
}

func (r *listReceiver) OnMessage(c netframe.Conn, b []byte) {
	log.Infof("recv msg %v", b)
	skyline.Send(context.Background(), gateSvcName, "ConnMessage", c, b)
}

func (r *listReceiver) OnClosed(c netframe.Conn) error {
	skyline.Send(context.Background(), gateSvcName, "CloseConn", c)
	return nil
}

var gateAddr = "0.0.0.0:8808"
var gateSvcName = "gate"

func main() {
	skyline.Init("config.json", func(ctx context.Context) error {
		_, err := skyline.NewService(gateSvcName, skeleton.NewModule(&GateModule{}, nil), 0)
		if err != nil {
			log.Fatalf("new gate svc failed: %v", err)
		}
		return nil
	})
	lis, err := netframe.NewListener(gateAddr, func() netframe.Receiver {
		return &listReceiver{}
	})
	if err != nil {
		log.Fatalf("listen gate port failed: %v", err)
	}
	go func() {
		err = lis.Serve()
		if err != nil {
			log.Fatalf("gate listener serve err: %v", err)
		} else {
			log.Warning("gate listener quit serve")
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT)
	<-c

	skyline.Exit()
}
