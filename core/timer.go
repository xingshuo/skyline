package core

import (
	"container/list"
	"context"
	"time"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline/proto"
)

const perTickOverload = 1024

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
	service       *Service
	timers        map[uint32]*Timer
	seqSlots      map[time.Duration]*list.List
	debtFrameTime int64
}

func (tp *TimerPool) Init(s *Service, tickPrecision time.Duration) {
	tp.service = s
	tp.timers = make(map[uint32]*Timer)
	tp.seqSlots = make(map[time.Duration]*list.List)
	tp.debtFrameTime = int64(tickPrecision) / 2
}

func (tp *TimerPool) Release() {
	tp.timers = make(map[uint32]*Timer)
	tp.seqSlots = make(map[time.Duration]*list.List)
}

func (tp *TimerPool) NewGoTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
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

func (tp *TimerPool) NewFrameTimer(callOut TimerFunc, interval time.Duration, count int) uint32 {
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

func pcall(ctx context.Context, t *Timer) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Process timer function error, TimerID=%d, Error=%v", t.ID, err)
		}
	}()

	t.callOut(ctx)
}

func (tp *TimerPool) checkFrameTimers(ctx context.Context, now int64) int {
	execNum := 0
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
			execNum++
			pcall(ctx, t)
		}
	}

	return execNum
}

func (tp *TimerPool) OnTick(ctx context.Context) {
	execNum := 0
	now := time.Now().UnixNano()
	if tp.debtFrameTime > 0 {
		execNum = tp.checkFrameTimers(ctx, now-tp.debtFrameTime)
	}
	execNum += tp.checkFrameTimers(ctx, now)
	if execNum >= perTickOverload {
		log.Warningf("May overload, timer num = %d", execNum)
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
				time.AfterFunc(t.interval, func() {
					tp.service.PushMsg(0, proto.PTYPE_TIMER, id)
				})
			}
		} else {
			time.AfterFunc(t.interval, func() {
				tp.service.PushMsg(0, proto.PTYPE_TIMER, id)
			})
		}
		t.callOut(ctx)
	} else { // maybe stopped
		log.Debugf("unknown timer id %d", id)
	}
}
