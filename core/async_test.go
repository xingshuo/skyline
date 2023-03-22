package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xingshuo/skyline/defines"

	"github.com/xingshuo/skyline/log"
)

func TestAsyncPool_Go(t *testing.T) {
	log.Init("", log.LevelInfo)

	app := &Server{
		handleServices: make(map[defines.SVC_HANDLE]*Service),
		nameServices:   make(map[string]*Service),
		handleIndex:    0,
	}
	svc, err := app.NewService("test_async_go", &defines.DummyModule{}, 0)
	assert.Equal(t, nil, err)
	assert.NotNil(t, svc)

	quitCh := make(chan int)
	retVal := 100
	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		svc.Go(ctx, func() (interface{}, error) {
			time.Sleep(3 * time.Second)
			return retVal, nil
		}, func(ctx context.Context, reply interface{}, err error) {
			assert.Equal(t, retVal, reply)
			assert.Nil(t, err)
			quitCh <- retVal
		})
	})

	ticker := time.NewTicker(time.Second)
	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++
			log.Infof("tick %d", counter)
		case r := <-quitCh:
			assert.Equal(t, retVal, r)
			ticker.Stop()
			counter = -1
			log.Info("stop tick")
		}
		if counter < 0 {
			break
		}
	}

	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		ctx = context.WithValue(ctx, defines.CtxKeyRpcTimeout, time.Second)
		svc.Go(ctx, func() (interface{}, error) {
			time.Sleep(3 * time.Second)
			return retVal, nil
		}, func(_ context.Context, reply interface{}, err error) {
			assert.Nil(t, reply)
			assert.Equal(t, defines.ErrRpcTimeout, err)
			quitCh <- retVal * 2
		})
	})
	r := <-quitCh
	assert.Equal(t, retVal*2, r)

	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		svc.Go(ctx, func() (interface{}, error) {
			quitCh <- retVal * 4
			return nil, nil
		}, nil)
	})
	r = <-quitCh
	assert.Equal(t, retVal*4, r)
}

func TestAsyncPool_LinearGo(t *testing.T) {
	app := &Server{
		handleServices: make(map[defines.SVC_HANDLE]*Service),
		nameServices:   make(map[string]*Service),
		handleIndex:    0,
	}
	svc, err := app.NewService("test_async_go", &defines.DummyModule{}, 0)
	assert.Nil(t, err)
	assert.NotNil(t, svc)
	ch := make(chan struct{})
	svc.Spawn(func(ctx context.Context, _ ...interface{}) {
		svc.LinearGo(ctx, func() (interface{}, error) {
			log.Info("linear req func1 begin")
			time.Sleep(1 * time.Second)
			log.Info("linear req func1 done")
			return 100, nil
		}, func(ctx context.Context, reply interface{}, err error) {
			assert.Equal(t, 100, reply)
			assert.Nil(t, err)
			log.Info("linear cb func1 executed")
		})

		svc.LinearGo(ctx, func() (interface{}, error) {
			log.Info("linear req func2 begin")
			time.Sleep(2 * time.Second)
			log.Info("linear req func2 done")
			return 200, nil
		}, func(ctx context.Context, reply interface{}, err error) {
			log.Info("linear cb func2 execute begin")
			assert.Nil(t, err)
			assert.Equal(t, 200, reply)
			log.Info("linear cb func2 executed")
		})

		svc.LinearGo(ctx, func() (interface{}, error) {
			log.Info("linear req func3 begin")
			time.Sleep(2 * time.Second)
			log.Info("linear req func3 done")
			return 300, nil
		}, func(ctx context.Context, reply interface{}, err error) {
			assert.Nil(t, err)
			assert.Equal(t, 300, reply)
			log.Info("linear cb func3 executed")
		})

		ctx = context.WithValue(ctx, defines.CtxKeyRpcTimeout, 2*time.Second)
		svc.LinearGo(ctx, func() (interface{}, error) {
			log.Info("linear req func4 begin")
			time.Sleep(4 * time.Second)
			log.Info("linear req func4 done")
			return 400, nil
		}, func(ctx context.Context, reply interface{}, err error) {
			assert.Equal(t, defines.ErrRpcTimeout, err)
			assert.Nil(t, reply)
			log.Info("linear cb func4 executed")
			ch <- struct{}{}
		})
	})

	<-ch
}
