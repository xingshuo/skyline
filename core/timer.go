package core

import (
	"container/list"
	"context"
	"time"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline/proto"
)

type TimerFunc func(ctx context.Context)
type Timer struct {
	ID       uint32
	callOut  TimerFunc
	count    int
	interval time.Duration
	createAt int64
	elapse   int64
	seqIter  *list.Element
}

type TimerPool struct {
	service  *Service
	timers   map[uint32]*Timer
	seqSlots map[time.Duration]*list.List
}

func (tp *TimerPool) Init(s *Service) {
	tp.service = s
	tp.timers = make(map[uint32]*Timer)
	tp.seqSlots = make(map[time.Duration]*list.List)
}

func (tp *TimerPool) Release() {
	tp.timers = make(map[uint32]*Timer)
	tp.seqSlots = make(map[time.Duration]*list.List)
}

func (tp *TimerPool) NewTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
	if interval <= 0 {
		log.Warning("new timer interval <= 0, convert to Spawn")
		tp.service.Spawn(func(ctx context.Context, _ ...interface{}) {
			callOut(ctx)
		})
		return 0
	}
	id := tp.service.NewSession()
	t := &Timer{
		ID:       id,
		callOut:  callOut,
		count:    count,
		interval: interval,
		createAt: time.Now().UnixNano(),
		elapse:   int64(interval),
	}
	tp.timers[id] = t
	time.AfterFunc(t.interval, func() {
		tp.service.PushMsg(0, proto.PTYPE_TIMER, id)
	})
	return id
}

func (tp *TimerPool) NewSeqTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
	if interval <= 0 {
		log.Warning("new seq timer interval <= 0, convert to Spawn")
		tp.service.Spawn(func(ctx context.Context, _ ...interface{}) {
			callOut(ctx)
		})
		return 0
	}
	id := tp.service.NewSession()
	t := &Timer{
		ID:       id,
		callOut:  callOut,
		count:    count,
		interval: interval,
		createAt: time.Now().UnixNano(),
		elapse:   int64(interval),
	}
	tp.timers[id] = t

	slot := tp.seqSlots[interval]
	if slot == nil {
		slot = list.New()
		tp.seqSlots[interval] = slot
	}
	t.seqIter = slot.PushBack(t)
	return id
}

func (tp *TimerPool) StopTimer(id uint32) bool {
	t := tp.timers[id]
	if t != nil {
		delete(tp.timers, id)
		if t.seqIter != nil {
			slot := tp.seqSlots[t.interval]
			if slot != nil {
				slot.Remove(t.seqIter)
			}
		}
		return true
	}
	return false
}

func (tp *TimerPool) OnTick(ctx context.Context) {
	now := time.Now().UnixNano()
	for interval, slot := range tp.seqSlots {
		for {
			e := slot.Front()
			if e == nil {
				delete(tp.seqSlots, interval)
				break
			}
			t := e.Value.(*Timer)
			if t.createAt+t.elapse > now {
				break
			}
			if t.count > 0 {
				t.count--
				if t.count == 0 {
					slot.Remove(e)
					delete(tp.timers, t.ID)
				} else {
					t.elapse += int64(t.interval)
					slot.MoveToBack(e)
				}
			} else {
				t.elapse += int64(t.interval)
				slot.MoveToBack(e)
			}
			t.callOut(ctx)
		}
	}
}

func (tp *TimerPool) OnTimeout(ctx context.Context, id uint32) {
	t := tp.timers[id]
	if t != nil {
		if t.count > 0 { // 有限次执行
			t.count--
			if t.count == 0 {
				delete(tp.timers, id)
			} else {
				time.AfterFunc(time.Duration(t.interval)*time.Millisecond, func() {
					tp.service.PushMsg(0, proto.PTYPE_TIMER, id)
				})
			}
		} else {
			time.AfterFunc(time.Duration(t.interval)*time.Millisecond, func() {
				tp.service.PushMsg(0, proto.PTYPE_TIMER, id)
			})
		}
		t.callOut(ctx)
	} else {
		log.Errorf("unknown timer id %d", id)
	}
}
