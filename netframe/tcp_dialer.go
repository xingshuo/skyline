package netframe

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/log"
)

// 基于Tcp向指定目标地址发包流程封装

type State int

func (s State) String() string {
	switch s {
	case Idle:
		return "IDLE"
	case Connecting:
		return "CONNECTING"
	case Ready:
		return "READY"
	case TransientFailure:
		return "TRANSIENT_FAILURE"
	case Shutdown:
		return "SHUTDOWN"
	default:
		return "Invalid-State"
	}
}

const (
	// Transport初始化状态
	Idle State = iota
	// 尝试连接中
	Connecting
	// 可正常发包
	Ready
	// 连接异常状态
	TransientFailure
	// Transport关闭
	Shutdown
)

type Transport struct {
	conn  *TCPConn
	state State
	rwMu  sync.Mutex
}

func (t *Transport) getConn(d *Dialer) (*TCPConn, error) {
	t.rwMu.Lock()
	defer t.rwMu.Unlock()
	if t.state == Ready {
		return t.conn, nil
	}
	if t.state == Shutdown {
		return nil, defines.ErrTransportClosed
	}
	timeoutSec := d.opts.dialTimeout
	if timeoutSec <= 0 {
		timeoutSec = MAX_DIAL_TIMEOUT_SEC
	}
	t.state = Connecting
	rawConn, err := net.DialTimeout("tcp", d.address, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		t.state = TransientFailure
		return nil, err
	}
	t.conn = &TCPConn{rawConn: rawConn}
	err = t.conn.Init(rawConn, d.newReceiver())
	if err != nil {
		t.state = TransientFailure
		return nil, err
	}
	t.state = Ready
	go func() {
		err := t.conn.loopRead()
		if err != nil {
			t.rwMu.Lock()
			t.conn.Close()
			t.state = TransientFailure
			t.rwMu.Unlock()
			log.Errorf("loop read err:%v", err)
		}
	}()
	go func() {
		err := t.conn.loopWrite()
		if err != nil {
			t.rwMu.Lock()
			t.conn.Close()
			t.state = TransientFailure
			t.rwMu.Unlock()
			log.Errorf("loop write err:%v", err)
		}
	}()
	return t.conn, nil
}

func (t *Transport) shutdown() error {
	t.rwMu.Lock()
	defer t.rwMu.Unlock()
	if t.state == Shutdown {
		return fmt.Errorf("already shutdown")
	}
	t.state = Shutdown
	if t.conn != nil {
		t.conn.Close()
	}
	return nil
}

type Dialer struct {
	opts        dialOptions
	address     string
	newReceiver func() Receiver
	transport   *Transport
}

//外部调用接口
func (d *Dialer) Start() error {
	_, err := d.transport.getConn(d)
	return err
}

func (d *Dialer) Shutdown() error {
	return d.transport.shutdown()
}

func (d *Dialer) Send(b []byte) error {
	conn, err := d.transport.getConn(d)
	if err != nil {
		return err
	}
	conn.Send(b)
	return nil
}
