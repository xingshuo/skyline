package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xingshuo/skyline/skeleton"

	"github.com/xingshuo/skyline/netframe"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline"
)

type listReceiver struct {
	gateSvc skyline.Service
}

func (r *listReceiver) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 2, true
}

func (r *listReceiver) OnConnected(c netframe.Conn) error {
	r.gateSvc.PostRequest("NewConn", c)
	return nil
}

func (r *listReceiver) OnMessage(c netframe.Conn, b []byte) {
	log.Infof("recv msg %v", b)
	r.gateSvc.PostRequest("ConnMessage", c, b)
}

func (r *listReceiver) OnClosed(c netframe.Conn) error {
	r.gateSvc.PostRequest("CloseConn", c)
	return nil
}

var gateAddr = "0.0.0.0:8808"

func main() {
	skyline.Init("config.json")
	gateSvc, err := skyline.NewService("gate", skeleton.NewModule(&GateModule{}, nil), 0)
	if err != nil {
		log.Fatalf("new gate svc failed: %v", err)
	}
	lis, err := netframe.NewListener(gateAddr, func() netframe.Receiver {
		return &listReceiver{gateSvc}
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
