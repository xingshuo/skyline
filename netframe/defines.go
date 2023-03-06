package netframe

const (
	MAX_DIAL_TIMEOUT_SEC = 6
)

//TCPConn 发包能力的的interface抽象
type Conn interface {
	Send(b []byte)
	PeerAddr() string //获取连接对端地址
}

//流事件接收器
type Receiver interface {
	//headerLen：包头长度(目前只支持2或4)
	//bigEndian：大端序or小端序，推荐大端序
	HeaderFormat() (headerLen int64, bigEndian bool)

	//连接建立后
	OnConnected(c Conn) error

	//处理消息
	OnMessage(c Conn, b []byte)

	//连接断开前
	OnClosed(c Conn) error
}

type DefaultReceiver struct {
}

func (r *DefaultReceiver) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 4, true
}

func (r *DefaultReceiver) OnConnected(c Conn) error {
	return nil
}

func (r *DefaultReceiver) OnMessage(c Conn, b []byte) {
}

func (r *DefaultReceiver) OnClosed(c Conn) error {
	return nil
}
