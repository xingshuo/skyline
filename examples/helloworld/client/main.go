package main

import (
	"encoding/binary"
	"flag"
	"log"
	"time"

	"github.com/xingshuo/skyline/utils"

	"github.com/xingshuo/skyline/netframe"
)

type Receiver struct {
	roleId uint64
}

func (r *Receiver) HeaderFormat() (headerLen int64, bigEndian bool) {
	return 4, true
}

func (r *Receiver) OnConnected(c netframe.Conn) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], r.roleId)
	msg := append(b[:], utils.String2Bytes("helloworld")...)
	bodyLen := len(msg)
	data := make([]byte, bodyLen+2)
	binary.BigEndian.PutUint16(data, uint16(bodyLen))
	copy(data[2:], msg)
	c.Send(data)
	return nil
}

func (r *Receiver) OnMessage(c netframe.Conn, b []byte) {
	log.Printf("role %d recv reply msg: %s", r.roleId, utils.Bytes2String(b))
}

func (r *Receiver) OnClosed(c netframe.Conn) error {
	return nil
}

func main() {
	var roleId uint64
	var addr string
	flag.Uint64Var(&roleId, "role_id", 1, "role id")
	if roleId == 0 {
		roleId = 1
	}
	flag.StringVar(&addr, "addr", "0.0.0.0:8808", "addr")
	flag.Parse()
	log.Printf("role %d start login", roleId)
	d, err := netframe.NewDialer(addr, func() netframe.Receiver {
		return &Receiver{roleId: roleId}
	})
	if err != nil {
		log.Fatalf("role %d connect failed: %v", roleId, err)
	}
	err = d.Start()
	if err != nil {
		log.Fatalf("role %d login failed: %v", roleId, err)
	}
	time.Sleep(time.Second * 2)
	log.Printf("role %d disconnect", roleId)
}
