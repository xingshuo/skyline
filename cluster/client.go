package cluster

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/xingshuo/skyline/config"

	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
)

type Client struct {
	clusterAddrs map[string]string           // clustername: address
	dialers      map[string]*netframe.Dialer // clustername: dialer
	rwMutex      sync.RWMutex
}

func (c *Client) Init() error {
	c.dialers = make(map[string]*netframe.Dialer)
	return c.Reload()
}

func (c *Client) Reload() error {
	confPath := config.ServerConf.MeshConfig
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		log.Errorf("read cluster config [%s] failed:%v.\n", confPath, err)
		return err
	}
	confAddrs := make(map[string]string)
	err = json.Unmarshal(data, &confAddrs)
	if err != nil {
		log.Errorf("load cluster config [%s] failed:%v.\n", confPath, err)
		return err
	}

	for cluster, addr := range c.clusterAddrs {
		if confAddrs[cluster] != addr {
			c.rwMutex.Lock()
			d := c.dialers[cluster]
			delete(c.dialers, cluster)
			c.rwMutex.Unlock()
			if d != nil {
				d.Shutdown()
			}
		}
	}
	c.clusterAddrs = confAddrs
	config.ClusterConf = confAddrs
	return nil
}

func (c *Client) GetDialer(clusterName string) (*netframe.Dialer, error) {
	addr, ok := c.clusterAddrs[clusterName]
	if !ok {
		return nil, fmt.Errorf("no such cluster %s", clusterName)
	}
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	d := c.dialers[clusterName]
	if d == nil {
		var err error
		d, err = netframe.NewDialer(addr, nil)
		if err != nil {
			return nil, err
		}
		err = d.Start()
		if err != nil {
			return nil, err
		}
		c.dialers[clusterName] = d
	}
	return d, nil
}

func (c *Client) Send(clusterName string, data []byte) error {
	d, err := c.GetDialer(clusterName)
	if err != nil {
		return err
	}
	return d.Send(data)
}

func (c *Client) Exit() {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	for _, d := range c.dialers {
		go d.Shutdown()
	}
	c.dialers = make(map[string]*netframe.Dialer)
}
