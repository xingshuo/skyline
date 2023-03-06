package lib

import (
	"sync"
	"sync/atomic"
)

// 确保事件只执行一次的原语封装,并提供通过channel是否读阻塞和bool类型两种判定事件是否执行的接口
type SyncEvent struct {
	fired       int32
	notifyFired chan struct{}
	fireOnce    sync.Once
}

func (e *SyncEvent) Fire() bool {
	ret := false
	e.fireOnce.Do(func() {
		atomic.StoreInt32(&e.fired, 1)
		close(e.notifyFired)
		ret = true
	})
	return ret
}

func (e *SyncEvent) Done() <-chan struct{} {
	return e.notifyFired
}

func (e *SyncEvent) HasFired() bool {
	return atomic.LoadInt32(&e.fired) == 1
}

func NewSyncEvent() *SyncEvent {
	return &SyncEvent{notifyFired: make(chan struct{})}
}
