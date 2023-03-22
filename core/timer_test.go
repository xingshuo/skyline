package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/log"
)

func TestService_NewGoTimer(t *testing.T) {
	log.Init("", log.LevelInfo)

	app := &Server{
		handleServices: make(map[defines.SVC_HANDLE]*Service),
		nameServices:   make(map[string]*Service),
		handleIndex:    0,
	}
	svc, err := app.NewService("test_gotimer", &defines.DummyModule{}, 0)
	assert.Equal(t, nil, err)
	assert.NotNil(t, svc)
	i := 0
	j := 0
	k := 0
	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		svc.NewGoTimer(func(ctx context.Context) {
			i++
		}, 700*time.Millisecond, 1)
		svc.NewGoTimer(func(ctx context.Context) {
			j++
		}, 300*time.Millisecond, 3)
		var handle uint32
		handle = svc.NewGoTimer(func(ctx context.Context) {
			k++
			if k >= 3 {
				svc.StopTimer(handle)
			}
		}, 400*time.Millisecond, 0)
		assert.NotEqual(t, 0, handle)
	})

	time.Sleep(550 * time.Millisecond)
	assert.Equal(t, 0, i)
	assert.Equal(t, 1, j)
	assert.Equal(t, 1, k)

	time.Sleep(550 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 2, k)

	time.Sleep(550 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 3, k)
}

func TestService_NewFrameTimer1(t *testing.T) {
	app := &Server{
		handleServices: make(map[defines.SVC_HANDLE]*Service),
		nameServices:   make(map[string]*Service),
		handleIndex:    0,
	}
	svc, err := app.NewService("test_timer", &defines.DummyModule{}, 500*time.Millisecond)
	assert.Equal(t, nil, err)
	assert.NotNil(t, svc)
	i := 0
	j := 0
	k := 0
	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		svc.NewTimer(func(ctx context.Context) {
			i++
		}, 700*time.Millisecond, 1)
		svc.NewTimer(func(ctx context.Context) {
			j++
		}, 300*time.Millisecond, 3)
		var handle uint32
		handle = svc.NewTimer(func(ctx context.Context) {
			k++
			if k >= 3 {
				svc.StopTimer(handle)
			}
		}, 400*time.Millisecond, 0)
		assert.NotEqual(t, 0, handle)
	})

	time.Sleep(550 * time.Millisecond)
	assert.Equal(t, 0, i)
	assert.Equal(t, 1, j)
	assert.Equal(t, 1, k)

	time.Sleep(550 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 2, k)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 2, k)

	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 3, k)

	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, 1, i)
	assert.Equal(t, 3, j)
	assert.Equal(t, 3, k)
}

func TestService_NewFrameTimer2(t *testing.T) {
	app := &Server{
		handleServices: make(map[defines.SVC_HANDLE]*Service),
		nameServices:   make(map[string]*Service),
		handleIndex:    0,
	}
	svc, err := app.NewService("test_timer", &defines.DummyModule{}, time.Second)
	assert.Equal(t, nil, err)
	assert.NotNil(t, svc)
	c := make(chan string, 6)
	svc.Spawn(func(ctx context.Context, args ...interface{}) {
		counter1 := 0
		svc.NewTimer(func(ctx context.Context) {
			counter1++
			c <- fmt.Sprintf("timer1-%d", counter1)
		}, 2*time.Second, 2)
		counter2 := 0
		svc.NewTimer(func(ctx context.Context) {
			counter2++
			c <- fmt.Sprintf("timer2-%d", counter2)
		}, 2*time.Second, 2)
		counter3 := 0
		svc.NewTimer(func(ctx context.Context) {
			counter3++
			c <- fmt.Sprintf("timer3-%d", counter3)
		}, 3*time.Second, 2)
	})

	orderList := []string{"timer1-1", "timer2-1", "timer3-1", "timer1-2", "timer2-2", "timer3-2"}
	for _, order := range orderList {
		v := <-c
		assert.Equal(t, order, v)
	}
}
