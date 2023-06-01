package skynet_cluster

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
	"github.com/xingshuo/skyline/seri"
	"math"
	"os"
	"sync"
)

const (
	MULTI_PART = 0x8000
)

type clusterSender struct {
	netframe.Dialer
	session uint32
}

func (cs *clusterSender) packPushRequest(address string, protoType int, msg []byte) (buf []byte, padding [][]byte) {
	sz := len(msg)
	nameLen := len(address)
	if sz < MULTI_PART {
		buf = make([]byte, sz+9+nameLen)
		binary.BigEndian.PutUint16(buf, uint16(sz+7+nameLen))
		buf[2] = 0x80
		buf[3] = uint8(nameLen)
		copy(buf[4:], address)
		binary.LittleEndian.PutUint32(buf[4+nameLen:], 0)
		buf[8+nameLen] = uint8(protoType)
		copy(buf[9+nameLen:], msg)
		return
	} else {
		buf = make([]byte, 13+nameLen)
		binary.BigEndian.PutUint16(buf, uint16(11+nameLen))
		buf[2] = 0xc1
		buf[3] = uint8(nameLen)
		copy(buf[4:], address)
		binary.LittleEndian.PutUint32(buf[4+nameLen:], cs.session)
		buf[8+nameLen] = uint8(protoType)
		binary.LittleEndian.PutUint32(buf[9+nameLen:], uint32(sz))
		cs.session++
		if cs.session > math.MaxInt32 {
			cs.session = 1
		}
		part := (sz-1)/MULTI_PART + 1
		offset := 0
		var tmpBuf []byte
		var s int
		for i := 0; i < part; i++ {
			if sz > MULTI_PART {
				s = MULTI_PART
				tmpBuf = make([]byte, s+7)
				tmpBuf[2] = 2
			} else {
				s = sz
				tmpBuf = make([]byte, s+7)
				tmpBuf[2] = 3 // the last multi part
			}
			binary.BigEndian.PutUint16(tmpBuf, uint16(s+5))
			binary.LittleEndian.PutUint32(tmpBuf[3:], cs.session)
			copy(tmpBuf[7:], msg[offset:offset+s])
			padding = append(padding, tmpBuf)
			sz -= s
			offset += s
		}
		return
	}
}

type skynetClient struct {
	confPath     string
	clusterAddrs map[string]string         // clustername: address
	senders      map[string]*clusterSender // clustername: sender
	rwMutex      sync.RWMutex
}

func (c *skynetClient) Init(confPath string) error {
	c.confPath = confPath
	c.senders = make(map[string]*clusterSender)
	return c.Reload()
}

func (c *skynetClient) Reload() error {
	data, err := os.ReadFile(c.confPath)
	if err != nil {
		log.Errorf("read cluster config [%s] failed:%v.\n", c.confPath, err)
		return err
	}
	confAddrs := make(map[string]string)
	err = json.Unmarshal(data, &confAddrs)
	if err != nil {
		log.Errorf("load cluster config [%s] failed:%v.\n", c.confPath, err)
		return err
	}

	for cluster, addr := range c.clusterAddrs {
		if confAddrs[cluster] != addr {
			c.rwMutex.Lock()
			s := c.senders[cluster]
			delete(c.senders, cluster)
			c.rwMutex.Unlock()
			if s != nil {
				s.Shutdown()
			}
		}
	}
	c.clusterAddrs = confAddrs
	return nil
}

func (c *skynetClient) GetSender(clusterName string) (*clusterSender, error) {
	addr, ok := c.clusterAddrs[clusterName]
	if !ok {
		return nil, fmt.Errorf("no such cluster %s", clusterName)
	}
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	s := c.senders[clusterName]
	if s == nil {
		d, err := netframe.NewDialer(addr, nil)
		if err != nil {
			return nil, err
		}
		err = d.Start()
		if err != nil {
			return nil, err
		}
		s = &clusterSender{
			Dialer:  *d,
			session: 1,
		}
		c.senders[clusterName] = s
	}
	return s, nil
}

func (c *skynetClient) Send(clusterName, svcName string, protoType int, args ...interface{}) error {
	s, err := c.GetSender(clusterName)
	if err != nil {
		return err
	}
	buf, padding := s.packPushRequest(svcName, protoType, seri.SeriPack(args))
	err = s.Send(buf)
	if err != nil {
		return err
	}
	for _, v := range padding {
		err = s.Send(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *skynetClient) Exit() {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	for _, s := range c.senders {
		go s.Shutdown()
	}
	c.senders = make(map[string]*clusterSender)
}
