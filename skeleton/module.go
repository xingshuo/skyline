package skeleton

import (
	"context"
	"errors"
)

type Processor interface {
	OnInit(ctx context.Context) error
	OnExit()
}

type Filter interface {
	BeforeProcess(context.Context, ...interface{}) ([]interface{}, error)
	AfterProcess(context.Context, interface{}, error) (interface{}, error)
}

type Module struct {
	handler_pool *HandlerPool
	processor    Processor
	filter       Filter
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
	if m.filter == nil {
		return m.handler_pool.PCall(ctx, args...)
	}

	ins, err := m.filter.BeforeProcess(ctx, args...)
	if err != nil {
		return nil, err
	}
	reply, err := m.handler_pool.PCall(ctx, ins...)
	return m.filter.AfterProcess(ctx, reply, err)
}

func (m *Module) RemoteProcess(ctx context.Context, args ...interface{}) (interface{}, error) {
	if m.filter == nil {
		return m.handler_pool.PCall(ctx, args...)
	}

	ins, err := m.filter.BeforeProcess(ctx, args...)
	if err != nil {
		return nil, err
	}
	reply, err := m.handler_pool.PCall(ctx, ins...)
	return m.filter.AfterProcess(ctx, reply, err)
}

func (m *Module) Exit() {
	m.processor.OnExit()
	m.handler_pool.OnExit()
}

func NewModule(pr Processor, fi Filter) *Module {
	return &Module{processor: pr, filter: fi}
}
