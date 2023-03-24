package skyline

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/xingshuo/skyline/cluster/codec"

	"github.com/xingshuo/skyline/proto"

	"github.com/xingshuo/skyline/interfaces"

	"github.com/xingshuo/skyline/defines"

	"github.com/xingshuo/skyline/config"
	"github.com/xingshuo/skyline/core"
	slog "github.com/xingshuo/skyline/log"
)

type Service interface {
	GetName() string
	GetHandle() defines.SVC_HANDLE
	Spawn(f core.SpawnFunc, args ...interface{})
}

var app *core.Server

// 框架初始化接口
func Init(confPath string, startFunc func(ctx context.Context) error) {
	if app != nil {
		log.Fatalln("app already init")
	}
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Fatalf("read config [%s] failed:%v\n", confPath, err)
	}
	err = json.Unmarshal(data, &config.ServerConf)
	if err != nil {
		log.Fatalf("load config [%s] failed:%v.\n", confPath, err)
	}
	slog.Init(config.ServerConf.LogFilename, slog.LogLevel(config.ServerConf.LogLevel))

	app = &core.Server{}
	err = app.Init()
	if err != nil {
		log.Fatalf("app init failed:%v", err)
	}
	bootstrap, err := app.NewService(defines.BootstrapSvcName, &defines.DummyModule{}, 0)
	if err != nil {
		log.Fatalf("new bootstrap service failed:%v", err)
	}
	wait := make(chan error, 1)
	bootstrap.Spawn(func(ctx context.Context, args ...interface{}) {
		wait <- startFunc(ctx)
	})
	err = <-wait
	app.DelService(defines.BootstrapSvcName)
	if err != nil {
		log.Fatalf("bootstrap failed:%v", err)
	}
	slog.Info("Init done")
}

// 框架退出接口
func Exit() {
	if app != nil {
		app.Exit()
	}
}

// 创建Service接口(goroutine safe)
func NewService(svcName string, module interfaces.Module, tickPrecision time.Duration) (Service, error) {
	return app.NewService(svcName, module, tickPrecision)
}

// 获取Service实例接口(goroutine safe)
func GetService(svcName string) Service {
	return app.GetService(svcName)
}

// 销毁Service接口(goroutine safe)
func DelService(svcName string) {
	app.DelService(svcName)
}

// 获取当前运行的service(goroutine safe)
func RunningService(ctx context.Context) Service {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	return svc
}

// 功能: 基于time.AfterFunc封装的定时器,保证callOut在Service内以消息通知的方式被执行
// 入参:
//	   callOut: 回调函数
//	   interval: 执行间隔, <= 0时, 会自动转化为Spawn调用
// 	   count: 执行次数, > 0:有限次, == 0:无限次
// 出参:
//	   定时器句柄, 可用于取消(StopTimer)
func NewGoTimer(ctx context.Context, callOut core.TimerFunc, interval time.Duration, count int) uint32 {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run NewGoTimer in unsafe goroutine")
	}
	return svc.NewGoTimer(callOut, interval, count)
}

// 功能: 当Service启动固定频率的跳帧时(NewService传入tickPrecision > 0), 会在每帧OnTick时检测其是否触发(通常会损失一定精度)
//	   否则转化为NewGoTimer
// 入参/出参:
//	   同上
func NewTimer(ctx context.Context, callOut core.TimerFunc, interval time.Duration, count int) uint32 {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run NewTimer in unsafe goroutine")
	}
	return svc.NewTimer(callOut, interval, count)
}

// 移除定时器
func StopTimer(ctx context.Context, seq uint32) bool {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run StopTimer in unsafe goroutine")
	}
	return svc.StopTimer(seq)
}

// 将f和args以一条消息的形式Push到当前Service消息队列中,等待被执行(goroutine safe)
func Spawn(ctx context.Context, f core.SpawnFunc, args ...interface{}) {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run Spawn in unsafe goroutine")
	}
	svc.Spawn(f, args...)
}

// 将f和args以一条消息的形式Push到目标Service消息队列中,等待被执行(goroutine safe)
func SpawnTo(svcName string, f core.SpawnFunc, args ...interface{}) error {
	ds := app.GetService(svcName)
	if ds == nil {
		return fmt.Errorf("unknown dst svc %s", svcName)
	}
	ds.Spawn(f, args...)
	return nil
}

// Service间投递消息接口(goroutine safe)
func Send(ctx context.Context, svcName string, args ...interface{}) error {
	ds := app.GetService(svcName)
	if ds == nil {
		return fmt.Errorf("unknown dst svc %s", svcName)
	}
	source := defines.SVC_HANDLE(0)
	srcSvc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if srcSvc != nil {
		source = srcSvc.GetHandle()
	}
	ds.PushMsg(source, proto.PTYPE_REQUEST, 0, args...)
	return nil
}

// Cluster间投递消息接口(goroutine safe)
func SendRemote(ctx context.Context, clusterName, svcName string, args ...interface{}) error {
	localCluster := config.ServerConf.ClusterName
	request, encErr := app.GetRpcCodec().EncodeRequest(args...)
	if encErr != nil {
		return encErr
	}
	localSvc := ""
	srcSvc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if srcSvc != nil {
		localSvc = srcSvc.GetName()
	}
	data, err := codec.PackClusterRequest(localCluster, localSvc, svcName, 0, request)
	if err != nil {
		return err
	}
	return app.GetRpcClient().Send(clusterName, data)
}

// Service间异步rpc接口, 目前没有超时处理
func AsyncCall(ctx context.Context, cb core.AsyncCbFunc, svcName string, args ...interface{}) error {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run AsyncCall in unsafe goroutine")
	}
	ds := app.GetService(svcName)
	if ds == nil {
		return fmt.Errorf("unknown dst svc %s", svcName)
	}
	return svc.GetAsyncPool().AsyncCall(ctx, ds, args, cb)
}

// Cluster间异步rpc接口, 可以通过context.WithValue(ctx, defines.CtxKeyRpcTimeout, xxxx)指定rpc超时时长(默认6s)
func AsyncCallRemote(ctx context.Context, cb core.AsyncCbFunc, clusterName, svcName string, args ...interface{}) error {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run AsyncCallRemote in unsafe goroutine")
	}
	if cb == nil {
		log.Fatal("run AsyncCallRemote without cb, Maybe use AsyncCall instead")
	}
	timeout, _ := ctx.Value(defines.CtxKeyRpcTimeout).(time.Duration)
	if timeout <= 0 {
		timeout = defines.DefaultSSRpcTimeout
	}
	return svc.GetAsyncPool().AsyncCallRemote(clusterName, svcName, args, cb, timeout)
}

// 基于Service的异步并发接口: 通过单独的goroutine执行f, 执行完成后以服务消息的方式通知并执行cb
func Go(ctx context.Context, f core.GoReqFunc, cb core.AsyncCbFunc) {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run Go in unsafe goroutine")
	}
	svc.Go(ctx, f, cb)
}

// 基于Service的异步线性并发接口: 通过单独的goroutine执行f,且所有处理f的goroutine按队列序执行. 当某个f执行完成后,立即以服务消息的方式通知其对应cb执行
func LinearGo(ctx context.Context, f core.GoReqFunc, cb core.AsyncCbFunc) {
	svc, _ := ctx.Value(defines.CtxKeyService).(*core.Service)
	if svc == nil {
		log.Fatal("run LinearGo in unsafe goroutine")
	}
	svc.LinearGo(ctx, f, cb)
}
