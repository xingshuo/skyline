package lib

// 单链表

type listNode struct {
	it   interface{}
	next *listNode
}

type List struct {
	head *listNode
	tail *listNode
	len  int
}

//nolint
func (il *List) Init() *List {
	il.head, il.tail = nil, nil
	il.len = 0
	return il
}

func (il *List) IsEmpty() bool {
	return il.head == nil
}

func (il *List) Len() int {
	return il.len
}

func (il *List) Enqueue(i interface{}) {
	n := &listNode{it: i}
	if il.tail == nil {
		il.head, il.tail = n, n
		il.len++
		return
	}
	il.tail.next = n
	il.tail = n
	il.len++
}

func (il *List) Peek() interface{} {
	return il.head.it
}

//nolint
func (il *List) Dequeue() interface{} {
	if il.head == nil {
		return nil
	}
	i := il.head.it
	il.head = il.head.next
	if il.head == nil {
		il.tail = nil
	}
	il.len--
	return i
}

func NewList() *List {
	return &List{}
}
