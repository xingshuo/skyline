package skeleton

import (
	"context"
	"errors"
)

type Processor interface {
	OnInit(ctx context.Context) error
	OnExit()
}

type Interceptor interface {
	BeforeProcess(context.Context, ...interface{}) ([]interface{}, error)
	AfterProcess(context.Context, interface{}, error) (interface{}, error)
}

type Module struct {
	handler_pool *HandlerPool
	processor    Processor
	interceptor  Interceptor
}

func (m *Module) Init(ctx context.Context) error {
	if m.processor == nil {
		return errors.New("skeleton module no register handlers")
	}
	err := m.processor.OnInit(ctx)
	if err != nil {
		return err
	}
	m.handler_pool = &HandlerPool{}
	err = m.handler_pool.Register(ctx, m.processor)
	if err != nil {
		return err
	}
	return nil
}

func (m *Module) LocalProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	if m.interceptor == nil {
		return m.handler_pool.Process(ctx, args...)
	}

	ins, err := m.interceptor.BeforeProcess(ctx, args...)
	if err != nil {
		return nil, err
	}
	reply, err := m.handler_pool.Process(ctx, ins...)
	return m.interceptor.AfterProcess(ctx, reply, err)
}

func (m *Module) RemoteProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	if m.interceptor == nil {
		return m.handler_pool.Process(ctx, args...)
	}

	ins, err := m.interceptor.BeforeProcess(ctx, args...)
	if err != nil {
		return nil, err
	}
	reply, err := m.handler_pool.Process(ctx, ins...)
	return m.interceptor.AfterProcess(ctx, reply, err)
}

func (m *Module) Exit() {
	m.processor.OnExit()
	m.handler_pool.OnExit()
}

func NewModule(p Processor, i Interceptor) *Module {
	return &Module{processor: p, interceptor: i}
}
