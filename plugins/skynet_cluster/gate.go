package skynet_cluster

import (
	"context"
	"encoding/binary"
	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
	"github.com/xingshuo/skyline/seri"
	"github.com/xingshuo/skyline/utils"
)

type clusterRequest struct {
	Address   string
	ProtoType int
	Msg       []byte
}

type clusterReqPadding struct {
	msgSz int
	req   *clusterRequest
}

type clusterAgent struct {
	largeRequests map[uint32]*clusterReqPadding
}

func (a *clusterAgent) unpackPushRequest(buf []byte) *clusterRequest {
	sz := len(buf)
	if sz < 2 {
		log.Errorf("Invalid cluster message (size=%d)", sz)
		return nil
	}
	switch buf[0] {
	case 0x80:
		nameSz := int(buf[1])
		if sz < nameSz+7 {
			log.Errorf("Invalid cluster message (size=%d)", sz)
			return nil
		}
		address := string(buf[2 : 2+nameSz])
		session := binary.LittleEndian.Uint32(buf[2+nameSz:])
		utils.Assert(session == 0, "Invalid req package. session != 0")
		protoType := int(buf[2+nameSz+4])
		clusReq := &clusterRequest{
			Address:   address,
			ProtoType: protoType,
			Msg:       buf[nameSz+7:],
		}
		return clusReq
	case 0xc1: // unpackmreq_head
		nameSz := int(buf[1])
		if sz < nameSz+11 {
			log.Errorf("Invalid cluster message. (size=%d)", sz)
			return nil
		}
		address := string(buf[2 : 2+nameSz])
		session := binary.LittleEndian.Uint32(buf[2+nameSz:])
		protoType := int(buf[2+nameSz+4])
		msgSz := binary.LittleEndian.Uint32(buf[nameSz+7:])
		utils.Assert(a.largeRequests[session] == nil, "Invalid large req package. session padding not nil")
		clusReq := &clusterRequest{
			Address:   address,
			ProtoType: protoType,
			Msg:       make([]byte, 0),
		}
		a.largeRequests[session] = &clusterReqPadding{
			msgSz: int(msgSz),
			req:   clusReq,
		}
		return nil
	case 2, 3: // unpackmreq_part
		if sz < 5 {
			log.Error("Invalid cluster multi part message")
			return nil
		}
		isEnd := (buf[0] == 3)
		session := binary.LittleEndian.Uint32(buf[1:])
		padding := a.largeRequests[session]
		utils.Assert(padding != nil, "Invalid large req multi part message. session padding is nil")
		padding.req.Msg = append(padding.req.Msg, buf[5:]...)
		if isEnd {
			delete(a.largeRequests, session)
			utils.Assert(padding.msgSz == len(padding.req.Msg), "Invalid large req, check msg size error")
			return padding.req
		} else {
			return nil
		}
	default:
		log.Errorf("Invalid req package type %v", buf[0])
	}
	return nil
}

func (a *clusterAgent) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 2, true
}

func (a *clusterAgent) OnConnected(c netframe.Conn) error {
	return nil
}

func (a *clusterAgent) OnMessage(c netframe.Conn, data []byte) {
	req := a.unpackPushRequest(data)
	if req != nil {
		skyline.Send(context.Background(), req.Address, seri.SeriUnpack(req.Msg)...)
	}
}

func (a *clusterAgent) OnClosed(s netframe.Conn) error {
	return nil
}

type skynetServer struct {
	gates map[string]*netframe.Listener
}

func (s *skynetServer) Listen(cluster string) error {
	lisAddr := client.clusterAddrs[cluster]
	if lisAddr == "" {
		log.Fatalf("skynet cluster no config: %s", cluster)
	}
	if s.gates[cluster] != nil {
		log.Errorf("skynet cluster listen again: %s", cluster)
		return defines.ErrAddrListenAgain
	}
	l, err := netframe.NewListener(lisAddr, func() netframe.Receiver {
		return &clusterAgent{
			largeRequests: make(map[uint32]*clusterReqPadding),
		}
	})
	if err != nil {
		return err
	}
	s.gates[cluster] = l
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

func (s *skynetServer) Exit() {
	for _, lis := range s.gates {
		lis.GracefulStop()
	}
}
