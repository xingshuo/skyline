package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xingshuo/skyline/cluster/codec"

	"github.com/xingshuo/skyline/interfaces"

	"github.com/xingshuo/skyline/cluster"

	"github.com/xingshuo/skyline/config"

	"github.com/xingshuo/skyline/proto"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/lib"
	"github.com/xingshuo/skyline/log"
)

type SpawnFunc func(ctx context.Context, args ...interface{})

type Service struct {
	server       *Server
	name         string // 服务名
	handle       defines.SVC_HANDLE
	ctx          context.Context
	mqueue       *MsgQueue
	msgNotify    chan struct{}
	exitNotify   *lib.SyncEvent
	exitDone     *lib.SyncEvent
	timerPool    *TimerPool
	asyncPool    *AsyncPool
	sessionIndex uint32
	module       interfaces.Module
	ticker       *time.Ticker
}

func (s *Service) Init(module interfaces.Module, tickPrecision time.Duration) error {
	s.ctx = context.WithValue(context.Background(), defines.CtxKeyService, s)
	s.mqueue = NewMQueue(defines.DefaultMQSize)
	s.msgNotify = make(chan struct{}, 1)
	s.exitNotify = lib.NewSyncEvent()
	s.exitDone = lib.NewSyncEvent()
	s.timerPool = &TimerPool{}
	s.timerPool.Init(s, tickPrecision)
	s.asyncPool = &AsyncPool{}
	s.asyncPool.Init(s)
	s.module = module
	if tickPrecision > 0 {
		s.ticker = time.NewTicker(tickPrecision)
	} else {
		log.Warningf("%s init without ticker", s)
	}
	return s.module.Init(s.ctx)
}

func (s *Service) String() string {
	return fmt.Sprintf("[%s-%d]", s.name, s.handle)
}

func (s *Service) GetName() string {
	return s.name
}

func (s *Service) GetContext() context.Context {
	return s.ctx
}

func (s *Service) GetHandle() defines.SVC_HANDLE {
	return s.handle
}

func (s *Service) GetRpcClient() *cluster.Client {
	return s.server.rpcClient
}

func (s *Service) GetRpcCodec() interfaces.Codec {
	return s.server.rpcCodec
}

func (s *Service) GetAsyncPool() *AsyncPool {
	return s.asyncPool
}

func (s *Service) NewSession() uint32 {
	s.sessionIndex++
	if s.sessionIndex == 0 {
		s.sessionIndex++
	}
	return s.sessionIndex
}

func (s *Service) PushMsg(source defines.SVC_HANDLE, msgType proto.MsgType, session uint32, data ...interface{}) {
	wakeUp := s.mqueue.Push(source, msgType, session, data)
	if wakeUp {
		select {
		case s.msgNotify <- struct{}{}:
		default:
		}
	}
}

func (s *Service) NewGoTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
	return s.timerPool.NewGoTimer(callOut, interval, count)
}

func (s *Service) NewTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
	if s.ticker == nil {
		return s.timerPool.NewGoTimer(callOut, interval, count)
	}
	return s.timerPool.NewFrameTimer(callOut, interval, count)
}

func (s *Service) StopTimer(seq uint32) bool {
	return s.timerPool.StopTimer(seq)
}

func (s *Service) Spawn(f SpawnFunc, args ...interface{}) {
	s.PushMsg(0, proto.PTYPE_SPAWN, 0, f, args)
}

func (s *Service) Go(ctx context.Context, f GoReqFunc, cb AsyncCbFunc) {
	s.asyncPool.Go(ctx, f, cb)
}

func (s *Service) LinearGo(ctx context.Context, f GoReqFunc, cb AsyncCbFunc) {
	s.asyncPool.LinearGo(ctx, f, cb)
}

func (s *Service) Serve() {
	log.Infof("cluster {%s} new service %s", config.ServerConf.ClusterName, s)
	if s.ticker != nil {
		for {
			select {
			case <-s.msgNotify:
				for {
					msg := s.mqueue.Pop()
					if msg == nil {
						break
					}
					s.dispatchMsg(msg.Source, msg.MsgType, msg.Session, msg.Data...)
				}
			case <-s.ticker.C:
				s.timerPool.OnTick(s.ctx)
			case <-s.exitNotify.Done():
				for {
					msg := s.mqueue.Pop()
					if msg == nil {
						break
					}
					s.dispatchMsg(msg.Source, msg.MsgType, msg.Session, msg.Data...)
				}
				s.exitDone.Fire()
				return
			}
		}
	} else {
		for {
			select {
			case <-s.msgNotify:
				for {
					msg := s.mqueue.Pop()
					if msg == nil {
						break
					}
					s.dispatchMsg(msg.Source, msg.MsgType, msg.Session, msg.Data...)
				}
			case <-s.exitNotify.Done():
				for {
					msg := s.mqueue.Pop()
					if msg == nil {
						break
					}
					s.dispatchMsg(msg.Source, msg.MsgType, msg.Session, msg.Data...)
				}
				s.exitDone.Fire()
				return
			}
		}
	}
}

func (s *Service) dispatchMsg(source defines.SVC_HANDLE, msgType proto.MsgType, session uint32, msg ...interface{}) {
	defer func() {
		if config.ServerConf.IsRecoverModel {
			if e := recover(); e != nil {
				log.Errorf("panic occurred on dispatch: %v, session:%v err: %v", msgType, session, e)
			}
		}
	}()
	log.Debugf("dispatch %v msg is %v", msgType, msg)

	switch msgType {
	case proto.PTYPE_REQUEST:
		srcSvc := s.server.GetServiceByHandle(source)
		ctx := s.ctx
		if srcSvc != nil {
			ctx = context.WithValue(ctx, defines.CtxKeySrcSvcName, srcSvc.GetName())
		}
		reply, err := s.module.LocalProcess(ctx, msg...)
		if session != 0 {
			if srcSvc != nil {
				srcSvc.PushMsg(s.handle, proto.PTYPE_RESPONSE, session, &proto.RpcResponse{
					Reply: reply,
					Err:   err,
				})
			} else {
				log.Errorf("unknown src service %d", source)
			}
		}
	case proto.PTYPE_CLUSTER_REQ:
		req := msg[0].(*proto.ClusterRequest)
		ctx := context.WithValue(s.ctx, defines.CtxKeySrcCluster, req.SrcCluster)
		ctx = context.WithValue(ctx, defines.CtxKeySrcSvcName, req.SrcService)
		reqArgs, decErr := s.GetRpcCodec().DecodeRequest(req.Request)
		if decErr != nil {
			log.Errorf("decode cluster request err:%v", decErr)
			return
		}
		reply, err := s.module.RemoteProcess(ctx, reqArgs...)
		if session != 0 {
			var errMsg string
			if err == nil {
				errMsg = defines.ErrOK.Error()
			} else {
				errMsg = err.Error()
			}
			localCluster := config.ServerConf.ClusterName
			response, encErr := s.GetRpcCodec().EncodeResponse(reply)
			if encErr != nil {
				log.Errorf("encode cluster response err:%v", encErr)
				return
			}
			data, packErr := codec.PackClusterResponse(localCluster, s.name, req.SrcService, session, response, errMsg)
			if packErr != nil {
				log.Errorf("pack cluster response err: %v", packErr)
				return
			}
			sendErr := s.server.rpcClient.Send(req.SrcCluster, data)
			if sendErr != nil {
				log.Errorf("send cluster response err: %v", sendErr)
			}
		}
	case proto.PTYPE_CLUSTER_RSP:
		rsp := msg[0].(*proto.ClusterResponse)
		reply, decErr := s.GetRpcCodec().DecodeResponse(rsp.Response)
		if decErr != nil {
			log.Errorf("decode cluster response err:%v", decErr)
			return
		}
		var err error
		if rsp.ErrMsg == defines.ErrOK.Error() {
			err = nil
		} else {
			err = errors.New(rsp.ErrMsg)
		}
		s.asyncPool.OnAsyncCb(s.ctx, session, reply, err)
	case proto.PTYPE_RESPONSE, proto.PTYPE_ASYNC_CB:
		rsp := msg[0].(*proto.RpcResponse)
		s.asyncPool.OnAsyncCb(s.ctx, session, rsp.Reply, rsp.Err)
	case proto.PTYPE_TIMER:
		s.timerPool.OnTimeout(s.ctx, session)
	case proto.PTYPE_SPAWN:
		f := msg[0].(SpawnFunc)
		args := msg[1].([]interface{})
		f(s.ctx, args...)
	}

	log.Debugf("%s dispatch %s done from %s", s, msgType, s.server.GetServiceByHandle(source))
}

func (s *Service) Exit() {
	if s.exitNotify.Fire() {
		<-s.exitDone.Done()
		s.module.Exit()
		s.timerPool.Release()
		if s.ticker != nil {
			s.ticker.Stop()
		}
	}
	log.Infof("service %s exit!\n", s)
}
