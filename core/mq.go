package core

import (
	"fmt"
	"sync"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/proto"
)

// 循环数组消息队列
func NewMQueue(cap int) *MsgQueue {
	mq := &MsgQueue{
		head:        0,
		tail:        0,
		cap:         cap,
		data:        make([]proto.Message, cap),
		waitConsume: true,
	}
	return mq
}

type MsgQueue struct {
	head        int // 队头
	tail        int // 队尾(指向下一个可放置位置)
	cap         int
	data        []proto.Message
	rwMu        sync.RWMutex
	waitConsume bool
}

func (mq *MsgQueue) expand() {
	newq := make([]proto.Message, mq.cap*2)
	for i := 0; i < mq.cap; i++ {
		newq[i] = mq.data[(mq.head+i)%mq.cap]
	}
	mq.data = newq
	mq.tail = mq.cap
	mq.head = 0
	mq.cap *= 2
}

func (mq *MsgQueue) Push(source defines.SVC_HANDLE, msgType proto.MsgType, session uint32, data []interface{}) bool {
	mq.rwMu.Lock()
	defer mq.rwMu.Unlock()
	wakeUp := mq.waitConsume
	if wakeUp {
		mq.waitConsume = false
	}
	back := &mq.data[mq.tail]
	back.Source = source
	back.MsgType = msgType
	back.Session = session
	back.Data = data
	mq.tail++
	if mq.tail >= mq.cap {
		mq.tail = 0
	}
	if mq.head == mq.tail {
		mq.expand()
	}
	return wakeUp
}

func (mq *MsgQueue) Pop() *proto.Message {
	mq.rwMu.Lock()
	defer mq.rwMu.Unlock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		mq.waitConsume = true
		return nil
	}
	top := &mq.data[mq.head]
	mq.head++
	if mq.head >= mq.cap {
		mq.head = 0
	}
	msg := &proto.Message{
		Source:  top.Source,
		MsgType: top.MsgType,
		Session: top.Session,
		Data:    top.Data,
	}
	return msg
}

func (mq *MsgQueue) Peek() *proto.Message {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	if mq.head == mq.tail { // 由于Push时相等会扩容,所以相等只可能是空
		return nil
	}
	top := &mq.data[mq.head]
	msg := &proto.Message{
		Source:  top.Source,
		MsgType: top.MsgType,
		Session: top.Session,
		Data:    top.Data,
	}
	return msg
}

func (mq *MsgQueue) Len() int {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	if mq.tail >= mq.head {
		return mq.tail - mq.head
	}
	return mq.tail - mq.head + mq.cap
}

func (mq *MsgQueue) debug() {
	mq.rwMu.RLock()
	defer mq.rwMu.RUnlock()
	fmt.Printf("head:%d tail:%d cap:%d len:%d\n", mq.head, mq.tail, mq.cap, mq.Len())
	for i := mq.head; i != mq.tail; i = (i + 1) % mq.cap {
		fmt.Println(mq.data[i])
	}
}
