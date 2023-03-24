package codec

import (
	"encoding/binary"
	"errors"

	"github.com/xingshuo/skyline/seri"

	s2sproto "github.com/xingshuo/skyline/proto/generate"

	"github.com/golang/protobuf/proto"
	"github.com/xingshuo/skyline/log"
)

//协议格式: 4字节包头长度 + 内容
const SSHeadLen = 4

func FillSSHeader(b []byte) []byte {
	bodyLen := len(b)
	data := make([]byte, bodyLen+SSHeadLen)
	binary.BigEndian.PutUint32(data, uint32(bodyLen))
	copy(data[SSHeadLen:], b)
	return data
}

func PackClusterRequest(srcCluster, srcSvc, dstSvc string, session uint32, request []byte) ([]byte, error) {
	req := &s2sproto.ReqClusterRequest{
		SrcCluster: srcCluster,
		SrcService: srcSvc,
		DstService: dstSvc,
		Session:    session,
		Request:    request,
	}
	ssmsg := &s2sproto.SSMsg{
		Cmd: s2sproto.SSCmd_REQ_CLUSTER_REQUEST,
		Msg: &s2sproto.SSMsg_ClusterRequest{
			ClusterRequest: req,
		},
	}
	b, err := proto.Marshal(ssmsg)
	if err != nil {
		log.Errorf("pb marshal err:%v.\n", err)
		return nil, err
	}
	data := FillSSHeader(b)
	return data, nil
}

func PackClusterResponse(srcCluster, srcSvc, dstSvc string, session uint32, response []byte, errMsg string) ([]byte, error) {
	rsp := &s2sproto.RspClusterResponse{
		SrcCluster: srcCluster,
		SrcService: srcSvc,
		DstService: dstSvc,
		Session:    session,
		Response:   response,
		ErrMsg:     errMsg,
	}
	ssmsg := &s2sproto.SSMsg{
		Cmd: s2sproto.SSCmd_RSP_CLUSTER_RESPONSE,
		Msg: &s2sproto.SSMsg_ClusterResponse{
			ClusterResponse: rsp,
		},
	}
	b, err := proto.Marshal(ssmsg)
	if err != nil {
		log.Errorf("pb marshal err:%v.\n", err)
		return nil, err
	}
	data := FillSSHeader(b)
	return data, nil
}

type LuaSeri struct {
}

func (ls *LuaSeri) EncodeRequest(args ...interface{}) ([]byte, error) {
	return seri.SeriPack(args...), nil
}

func (ls *LuaSeri) DecodeRequest(data []byte) ([]interface{}, error) {
	return seri.SeriUnpack(data), nil
}

func (ls *LuaSeri) EncodeResponse(reply interface{}) ([]byte, error) {
	return seri.SeriPack(reply), nil
}

func (ls *LuaSeri) DecodeResponse(data []byte) (interface{}, error) {
	l := seri.SeriUnpack(data)
	if len(l) != 1 {
		return nil, errors.New("lua seri decode len err")
	}
	return l[0], nil
}
