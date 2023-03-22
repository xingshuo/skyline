package core

import (
	"context"
	"sync"
	"time"

	"github.com/xingshuo/skyline/seri"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/proto"

	"github.com/xingshuo/skyline/config"
	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline/lib"
)

type GoReqFunc func() (interface{}, error)
type AsyncCbFunc func(ctx context.Context, reply interface{}, err error)

type AsyncPool struct {
	service     *Service
	cbFuncs     map[uint32]AsyncCbFunc
	linearQueue *lib.List
	linearMutex sync.Mutex
	execMutex   sync.Mutex
}

func (ap *AsyncPool) Init(s *Service) {
	ap.service = s
	ap.cbFuncs = make(map[uint32]AsyncCbFunc)
	ap.linearQueue = lib.NewList()
}

func (ap *AsyncPool) AsyncCall(ctx context.Context, dstSvc *Service, reqArgs []interface{}, cb AsyncCbFunc) error {
	seq := ap.service.NewSession()
	ap.cbFuncs[seq] = cb
	dstSvc.PushMsg(ap.service.GetHandle(), proto.PTYPE_REQUEST, seq, reqArgs...)
	return nil
}

func (ap *AsyncPool) AsyncCallRemote(clusterName, svcName string, reqArgs []interface{}, cb AsyncCbFunc, timeout time.Duration) error {
	localCluster := config.ServerConf.ClusterName
	seq := ap.service.NewSession()
	request := seri.SeriPack(reqArgs...)
	data, err := proto.PackClusterRequest(localCluster, ap.service.GetName(), svcName, seq, request)
	if err != nil {
		return err
	}
	err = ap.service.GetRpcClient().Send(clusterName, data)
	if err != nil {
		return err
	}
	var timerSeq uint32
	timerSeq = ap.service.NewTimer(func(ctx context.Context) {
		ap.OnAsyncCb(ctx, seq, nil, defines.ErrRpcTimeout)
	}, timeout, 1)
	ap.cbFuncs[seq] = func(ctx context.Context, reply interface{}, err error) {
		ap.service.StopTimer(timerSeq)
		cb(ctx, reply, err)
	}
	return nil
}

func (ap *AsyncPool) Go(ctx context.Context, f GoReqFunc, cb AsyncCbFunc) {
	var seq uint32
	var timeout time.Duration
	if cb != nil {
		seq = ap.service.NewSession()
		ap.cbFuncs[seq] = cb
		timeout, _ = ctx.Value(defines.CtxKeyRpcTimeout).(time.Duration)
		if timeout <= 0 {
			timeout = defines.DefaultSSRpcTimeout
		}
	}
	go func() {
		defer func() {
			if config.ServerConf.IsRecoverModel {
				if e := recover(); e != nil {
					log.Errorf("panic occurred on Go req func err: %v, seq: %v", e, seq)
				}
			}
		}()
		if seq == 0 {
			f()
			return
		}
		time.AfterFunc(timeout, func() {
			ap.service.PushMsg(0, proto.PTYPE_ASYNC_CB, seq, &proto.RpcResponse{
				Reply: nil,
				Err:   defines.ErrRpcTimeout,
			})
		})
		rsp, err := f()
		ap.service.PushMsg(0, proto.PTYPE_ASYNC_CB, seq, &proto.RpcResponse{
			Reply: rsp,
			Err:   err,
		})
	}()
}

type LinearPair struct {
	f       GoReqFunc
	cbSeq   uint32
	timeout time.Duration
}

func (ap *AsyncPool) LinearGo(ctx context.Context, f GoReqFunc, cb AsyncCbFunc) {
	var seq uint32
	var timeout time.Duration
	if cb != nil {
		seq = ap.service.NewSession()
		ap.cbFuncs[seq] = cb
		timeout, _ = ctx.Value(defines.CtxKeyRpcTimeout).(time.Duration)
		if timeout <= 0 {
			timeout = defines.DefaultSSRpcTimeout
		}
	}
	ap.linearMutex.Lock()
	ap.linearQueue.Enqueue(&LinearPair{
		f:       f,
		cbSeq:   seq,
		timeout: timeout,
	})
	ap.linearMutex.Unlock()
	go func() {
		ap.execMutex.Lock()
		defer ap.execMutex.Unlock()
		ap.linearMutex.Lock()
		pair := ap.linearQueue.Dequeue().(*LinearPair)
		ap.linearMutex.Unlock()
		defer func() {
			if config.ServerConf.IsRecoverModel {
				if e := recover(); e != nil {
					log.Errorf("panic occurred on LinearAsync req func err: %v, seq: %v", e, pair.cbSeq)
				}
			}
		}()
		if pair.cbSeq == 0 { // no callback
			pair.f()
			return
		}

		time.AfterFunc(pair.timeout, func() {
			ap.service.PushMsg(0, proto.PTYPE_ASYNC_CB, pair.cbSeq, &proto.RpcResponse{
				Reply: nil,
				Err:   defines.ErrRpcTimeout,
			})
		})
		rsp, err := pair.f()
		ap.service.PushMsg(0, proto.PTYPE_ASYNC_CB, pair.cbSeq, &proto.RpcResponse{
			Reply: rsp,
			Err:   err,
		})
	}()
}

func (ap *AsyncPool) OnAsyncCb(ctx context.Context, seq uint32, rsp interface{}, err error) {
	cb := ap.cbFuncs[seq]
	if cb != nil {
		delete(ap.cbFuncs, seq)
		cb(ctx, rsp, err)
	}
}
