package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/xingshuo/skyline/interfaces"

	"github.com/xingshuo/skyline/proto"

	"github.com/xingshuo/skyline/cluster"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline/defines"
)

type Server struct {
	rwMutex        sync.RWMutex
	handleServices map[defines.SVC_HANDLE]*Service
	nameServices   map[string]*Service
	handleIndex    uint64
	rpcClient      *cluster.Client
	rpcServer      *cluster.Server
}

func (s *Server) Init() error {
	s.handleServices = make(map[defines.SVC_HANDLE]*Service)
	s.nameServices = make(map[string]*Service)
	s.rpcClient = &cluster.Client{}
	err := s.rpcClient.Init()
	if err != nil {
		return err
	}
	s.rpcServer = &cluster.Server{}
	err = s.rpcServer.Init(s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Exit() {
	s.rpcClient.Exit()
	s.rpcServer.Exit()
	s.rwMutex.RLock()
	services := make([]string, 0, len(s.nameServices))
	for name := range s.nameServices {
		services = append(services, name)
	}
	s.rwMutex.RUnlock()
	// 顺序退出
	for _, name := range services {
		s.DelService(name)
	}
	log.Info("server exit!")
}

func (s *Server) newSvcHandle() uint64 {
	s.handleIndex++
	if s.handleIndex == 0 {
		s.handleIndex++
	}
	return s.handleIndex
}

func (s *Server) NewService(svcName string, module interfaces.Module, tickPrecision time.Duration) (*Service, error) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	svc := s.nameServices[svcName]
	if svc != nil {
		return nil, fmt.Errorf("service re-create")
	}
	handle := defines.SVC_HANDLE(s.newSvcHandle())
	svc = &Service{
		server: s,
		name:   svcName,
		handle: handle,
	}
	err := svc.Init(module, tickPrecision)
	if err != nil {
		log.Errorf("%s module init err: %v", svcName, err)
		return nil, err
	}
	s.handleServices[handle] = svc
	s.nameServices[svcName] = svc
	go svc.Serve()
	return svc, nil
}

func (s *Server) DelService(svcName string) {
	s.rwMutex.Lock()
	svc := s.nameServices[svcName]
	delete(s.nameServices, svcName)
	if svc != nil {
		delete(s.handleServices, svc.handle)
	}
	s.rwMutex.Unlock()
	if svc != nil {
		svc.Exit()
	}
}

func (s *Server) GetService(svcName string) *Service {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.nameServices[svcName]
}

func (s *Server) GetServiceByHandle(handle defines.SVC_HANDLE) *Service {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.handleServices[handle]
}

func (s *Server) PushToService(svcName string, source defines.SVC_HANDLE, msgType proto.MsgType, session uint32, data ...interface{}) error {
	s.rwMutex.RLock()
	svc := s.nameServices[svcName]
	s.rwMutex.RUnlock()
	if svc == nil {
		return defines.ErrSvcNoExist
	}
	svc.PushMsg(source, msgType, session, data...)
	return nil
}
