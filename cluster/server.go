package cluster

import (
	"github.com/xingshuo/skyline/proto"

	pbproto "github.com/golang/protobuf/proto"
	s2sproto "github.com/xingshuo/skyline/proto/generate"

	"github.com/xingshuo/skyline/config"
	"github.com/xingshuo/skyline/interfaces"
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
)

type Gate struct {
	app interfaces.App
}

func (g *Gate) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 4, true
}

func (g *Gate) OnConnected(c netframe.Conn) error {
	return nil
}

func (g *Gate) OnMessage(c netframe.Conn, data []byte) {
	var ssmsg s2sproto.SSMsg
	err := pbproto.Unmarshal(data, &ssmsg)
	if err != nil {
		log.Errorf("pb unmarshal err:%v.\n", err)
		return
	}
	if ssmsg.Cmd == s2sproto.SSCmd_REQ_CLUSTER_REQUEST {
		req := ssmsg.GetClusterRequest()
		reqBody := &proto.ClusterRequest{
			SrcCluster: req.SrcCluster,
			SrcService: req.SrcService,
			Request:    req.Request,
		}
		err = g.app.PushToService(req.DstService, 0, proto.PTYPE_CLUSTER_REQ, req.Session, reqBody)
		if err != nil {
			log.Errorf("cluster request not find dstSvc %s", req.DstService)
			return
		}
	} else if ssmsg.Cmd == s2sproto.SSCmd_RSP_CLUSTER_RESPONSE {
		rsp := ssmsg.GetClusterResponse()
		rspBody := &proto.ClusterResponse{
			SrcCluster: rsp.SrcCluster,
			SrcService: rsp.SrcService,
			Response:   rsp.Response,
			ErrMsg:     rsp.ErrMsg,
		}
		err = g.app.PushToService(rsp.DstService, 0, proto.PTYPE_CLUSTER_RSP, rsp.Session, rspBody)
		if err != nil {
			log.Errorf("cluster response not find dstSvc %s", rsp.DstService)
			return
		}
	}
}

func (g *Gate) OnClosed(s netframe.Conn) error {
	return nil
}

type Server struct {
	app interfaces.App
	lis *netframe.Listener
}

func (s *Server) Init(app interfaces.App) error {
	s.app = app
	localCluster := config.ServerConf.ClusterName
	lisAddr := config.ClusterConf[localCluster]
	if lisAddr == "" {
		log.Fatalf("no local cluster config: %v", localCluster)
	}
	l, err := netframe.NewListener(lisAddr, func() netframe.Receiver {
		return &Gate{app: app}
	})
	if err != nil {
		return err
	}
	s.lis = l
	log.Infof("cluster listen %s", lisAddr)
	go func() {
		err = l.Serve()
		if err != nil {
			log.Fatalf("gate listener serve err: %v", err)
		} else {
			log.Warning("gate listener quit serve")
		}
	}()
	return nil
}

func (s *Server) Exit() {
	s.lis.GracefulStop()
}
