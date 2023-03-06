package main

import (
	"context"
	"encoding/binary"

	"github.com/xingshuo/skyline"
	"github.com/xingshuo/skyline/defines"
	"github.com/xingshuo/skyline/log"
	"github.com/xingshuo/skyline/netframe"
	"github.com/xingshuo/skyline/utils"
)

type GateModule struct {
	roleId2conns map[uint64]netframe.Conn
	conn2roleIds map[netframe.Conn]uint64
	service      skyline.Service
}

// interface methods
func (g *GateModule) OnInit(ctx context.Context) error {
	g.roleId2conns = make(map[uint64]netframe.Conn)
	g.conn2roleIds = make(map[netframe.Conn]uint64)
	g.service = ctx.Value(defines.CtxKeyService).(skyline.Service)
	return nil
}

func (g *GateModule) OnExit() {
	g.roleId2conns = make(map[uint64]netframe.Conn)
	g.conn2roleIds = make(map[netframe.Conn]uint64)
}

// rpc handler methods
func (g *GateModule) NewConn(ctx context.Context, conn netframe.Conn) (interface{}, error) {
	g.addConn(conn, 0)
	return nil, nil
}

func (g *GateModule) ConnMessage(ctx context.Context, conn netframe.Conn, data []byte) (interface{}, error) {
	roleId := g.getRoleId(conn)
	uuid := binary.BigEndian.Uint64(data)
	if roleId == 0 {
		utils.Assert(uuid != 0, "invalid role id")
		g.addConn(conn, uuid)
		msg := utils.Bytes2String(data[8:])
		log.Infof("gate recv role %d first msg: %s", uuid, msg)
		reply := utils.String2Bytes("welcome to skyline server")
		bodyLen := len(reply)
		data := make([]byte, bodyLen+4)
		binary.BigEndian.PutUint32(data, uint32(bodyLen))
		copy(data[4:], reply)
		conn.Send(data)
	} else {
		utils.Assert(uuid == roleId, "invalid role id.")
		msg := utils.Bytes2String(data[8:])
		log.Infof("gate recv role %d msg: %s", uuid, msg)
	}
	return nil, nil
}

func (g *GateModule) CloseConn(ctx context.Context, conn netframe.Conn) (interface{}, error) {
	g.delConn(conn)
	return nil, nil
}

// inner methods
func (g *GateModule) addConn(conn netframe.Conn, uuid uint64) {
	g.conn2roleIds[conn] = uuid
	if uuid != 0 {
		g.roleId2conns[uuid] = conn
		log.Infof("register conn: %p, uuid: %v", conn, uuid)
	} else {
		log.Infof("new conn: %p", conn)
	}
}

func (g *GateModule) getRoleId(conn netframe.Conn) uint64 {
	return g.conn2roleIds[conn]
}

func (g *GateModule) delConn(conn netframe.Conn) {
	uuid := g.conn2roleIds[conn]
	delete(g.conn2roleIds, conn)
	if uuid != 0 {
		delete(g.roleId2conns, uuid)
	}
	log.Infof("del conn: %v, uuid: %v", conn, uuid)
}
