package skeleton

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/defines"

	"github.com/xingshuo/skyline/log"
)

var (
	typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
	typeOfContext = reflect.TypeOf(new(context.Context)).Elem()
)

type Handler struct {
	Method reflect.Method // method stub
}

type HandlerPool struct {
	Type     reflect.Type  // low-level type of method
	Receiver reflect.Value // receiver of method
	handlers map[string]*Handler
}

func isExported(name string) bool {
	w, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(w)
}

func isHandlerMethod(method reflect.Method) bool {
	mt := method.Type
	// Method must be exported.
	if method.PkgPath != "" {
		return false
	}
	mn := method.Name
	// Processor interface
	if mn == "OnInit" || mn == "OnExit" {
		return false
	}
	// Filter interface
	if mn == "BeforeProcess" || mn == "AfterProcess" {
		return false
	}

	if mt.NumIn() < 2 {
		return false
	}
	if t1 := mt.In(1); !t1.Implements(typeOfContext) {
		return false
	}

	if mt.NumOut() != 2 {
		return false
	}

	if mt.Out(1) != typeOfError {
		return false
	}

	return true
}

func (hp *HandlerPool) Register(ctx context.Context, p Processor) error {
	svc := ctx.Value(defines.CtxKeyService).(skyline.Service)
	hp.Type = reflect.TypeOf(p)
	hp.Receiver = reflect.ValueOf(p)
	hp.handlers = make(map[string]*Handler)

	typeName := reflect.Indirect(hp.Receiver).Type().Name()
	if typeName == "" {
		return errors.New("no service name for type " + hp.Type.String())
	}
	if !isExported(typeName) {
		return errors.New("type " + typeName + " is not exported")
	}

	for m := 0; m < hp.Type.NumMethod(); m++ {
		method := hp.Type.Method(m)
		if !isHandlerMethod(method) {
			continue
		}
		handler := &Handler{
			Method: method,
		}
		hp.handlers[method.Name] = handler
		log.Debugf("%s register handler: %s.%s", svc.GetName(), typeName, method.Name)
	}
	if len(hp.handlers) == 0 {
		return errors.New("type " + typeName + " has no exported methods of handler type")
	}
	return nil
}

func (hp *HandlerPool) PCall(ctx context.Context, args ...interface{}) (interface{}, error) {
	mn := args[0].(string)
	h := hp.handlers[mn]
	if h == nil {
		return nil, fmt.Errorf("no rpc handler %s", mn)
	}
	ins := make([]reflect.Value, len(args)+1)
	ins[0] = hp.Receiver
	ins[1] = reflect.ValueOf(ctx)
	for i, arg := range args[1:] {
		ins[2+i] = reflect.ValueOf(arg)
	}
	outs := h.Method.Func.Call(ins)
	var err error
	err, _ = outs[1].Interface().(error)
	return outs[0].Interface(), err
}

func (hp *HandlerPool) OnExit() {
	hp.Type = nil
	hp.Receiver = reflect.ValueOf(nil)
	hp.handlers = nil
}
