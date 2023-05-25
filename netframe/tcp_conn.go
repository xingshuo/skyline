package netframe

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/xingshuo/skyline/log"

	"github.com/xingshuo/skyline/lib"
)

// Tcp连接之上的封装层,主要处理网络流读写缓存相关

type TCPConn struct {
	rawConn         net.Conn
	wlist           *lib.List
	receiver        Receiver
	mu              sync.Mutex
	waiting         chan struct{}
	close           *lib.SyncEvent
	consumerWaiting bool
}

func (tc *TCPConn) Init(conn net.Conn, r Receiver) error {
	tc.rawConn = conn
	tc.wlist = lib.NewList()
	tc.receiver = r
	tc.waiting = make(chan struct{}, 1)
	tc.close = lib.NewSyncEvent()
	tc.consumerWaiting = false
	return r.OnConnected(tc)
}

func (tc *TCPConn) getNextMessage(headerLen int64, bigEndian bool) ([]byte, error) {
	header, err := io.ReadAll(io.LimitReader(tc.rawConn, headerLen))
	if err != nil {
		return nil, err
	}
	l := len(header)
	if l == 0 {
		return nil, errors.New("ErrConnectionClosed")
	}
	if l != int(headerLen) {
		return nil, errors.New("ErrInvalidHeader")
	}

	var byteOrder binary.ByteOrder
	if bigEndian {
		byteOrder = binary.BigEndian
	} else {
		byteOrder = binary.LittleEndian
	}
	var msgSize int64
	if headerLen == 4 {
		msgSize = int64(byteOrder.Uint32(header))
	} else { // headerLen == 2
		msgSize = int64(byteOrder.Uint16(header))
	}

	msgData, err := io.ReadAll(io.LimitReader(tc.rawConn, msgSize))
	if err != nil {
		return nil, err
	}
	if int64(len(msgData)) != msgSize {
		return nil, errors.New("ErrInvalidMsgData")
	}

	return msgData, nil
}

func (tc *TCPConn) loopRead() error {
	headerLen, bigEndian := tc.receiver.HeaderFormat()
	if headerLen != 2 && headerLen != 4 {
		panic(fmt.Errorf("Error receiver header len %d", headerLen))
	}
	for {
		msg, err := tc.getNextMessage(headerLen, bigEndian)
		if err != nil {
			log.Errorf("reading message err: %v", err)
			return err
		}
		tc.receiver.OnMessage(tc, msg)
	}
}

func (tc *TCPConn) loopWrite() error {
	var b []byte
	var err error
	for {
		tc.mu.Lock()
		if tc.wlist.IsEmpty() {
			tc.consumerWaiting = true
			tc.mu.Unlock()
			goto waitdata
		}
		b = tc.wlist.Dequeue().([]byte)
		tc.mu.Unlock()
		_, err = tc.rawConn.Write(b) //tcp conn once time drain off
		if err != nil {
			log.Errorf("tcp conn %p Write err: %v", tc, err)
			return err
		}
		continue

	waitdata:
		select {
		case <-tc.waiting:
		case <-tc.close.Done(): //连接断开通知
			log.Info("chan closed.\n")
			return nil
		}
	}
}

func (tc *TCPConn) Send(b []byte) {
	var wakeUp bool
	tc.mu.Lock()
	if tc.consumerWaiting {
		tc.consumerWaiting = false
		wakeUp = true
	}
	tc.wlist.Enqueue(b)
	tc.mu.Unlock()
	if wakeUp {
		select {
		case tc.waiting <- struct{}{}:
		default:
		}
	}
}

func (tc *TCPConn) Close() error { //主动关闭调用
	if tc.close.Fire() {
		tc.wlist.Init()
		tc.receiver.OnClosed(tc)
		tc.rawConn.Close()
		return nil
	}
	return fmt.Errorf("repeat close")
}

func (tc *TCPConn) Done() <-chan struct{} {
	return tc.close.Done()
}

func (tc *TCPConn) PeerAddr() string {
	return tc.rawConn.RemoteAddr().String()
}
