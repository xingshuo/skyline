package skyline

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"github.com/xingshuo/skyline/interfaces"

	"github.com/xingshuo/skyline/defines"

	"github.com/xingshuo/skyline/config"
	"github.com/xingshuo/skyline/core"
	slog "github.com/xingshuo/skyline/log"
)

type Service interface {
	GetName() string
	GetHandle() defines.SVC_HANDLE
	// 功能: 基于time.AfterFunc封装的定时器,保证callOut在Service内以消息通知的方式被执行
	// 入参:
	//	   callOut: 回调函数
	//	   interval: 执行间隔, <= 0时, 会自动转化为Spawn调用
	// 	   count: 执行次数, > 0:有限次, == 0:无限次
	// 出参:
	//	   定时器句柄, 可用于取消(StopTimer)
	NewGoTimer(callOut core.TimerFunc, interval time.Duration, count int) uint32
	// 功能: 当Service启动固定频率的跳帧时(NewService传入tickPrecision > 0), 会在每帧OnTick时检测其是否触发(通常会损失一定精度)
	//	   否则转化为NewGoTimer
	// 入参/出参:
	//	   同上
	NewTimer(callOut core.TimerFunc, interval time.Duration, count int) uint32
	StopTimer(seq uint32) bool
	Spawn(f core.SpawnFunc, args ...interface{})
	PostRequest(args ...interface{})
	Send(ctx context.Context, svcName string, args ...interface{}) error
	SendRemote(ctx context.Context, clusterName, svcName string, args ...interface{}) error
	AsyncCall(ctx context.Context, cb core.AsyncCbFunc, svcName string, args ...interface{}) error
	AsyncCallRemote(ctx context.Context, cb core.AsyncCbFunc, clusterName, svcName string, args ...interface{}) error
	Go(ctx context.Context, f core.GoReqFunc, cb core.AsyncCbFunc)
	LinearGo(ctx context.Context, f core.GoReqFunc, cb core.AsyncCbFunc)
}

var app *core.Server

func Init(confPath string) {
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
}

func Exit() {
	if app != nil {
		app.Exit()
	}
}

func NewService(svcName string, module interfaces.Module, tickPrecision time.Duration) (Service, error) {
	return app.NewService(svcName, module, tickPrecision)
}

func GetService(svcName string) Service {
	return app.GetService(svcName)
}

func DelService(svcName string) {
	app.DelService(svcName)
}
