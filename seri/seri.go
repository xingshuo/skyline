package seri

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"unsafe"
)

// Tips:所有>=2字节的整形, 统一按照大端序序列化和反序列化

// 低3位bits: 参数类型, 0~7
// 高5位bits: 具体类型对应子参数, 0~31
// 与lua/seri.c中定义对齐
const (
	_TYPE_NIL uint8 = 0

	// high bits : 0 false 1 true
	_TYPE_BOOLEAN uint8 = 1

	// high bits : TYPE_NUMBER_XXX
	_TYPE_NUMBER uint8 = 2

	// high bits : len
	_TYPE_SHORT_STRING uint8 = 4

	// high bits : len use bytes num
	_TYPE_LONG_STRING uint8 = 5

	_TYPE_TABLE uint8 = 6
)

const (
	_TYPE_NUMBER_ZERO  uint8 = 0
	_TYPE_NUMBER_BYTE  uint8 = 1
	_TYPE_NUMBER_WORD  uint8 = 2
	_TYPE_NUMBER_DWORD uint8 = 4
	_TYPE_NUMBER_QWORD uint8 = 6
	_TYPE_NUMBER_REAL  uint8 = 8
)

const (
	// 高5位bits容量上限+1
	_MAX_COOKIE = 32
	// 链式内存块节点,每块大小
	_BLOCK_SIZE = 128
	// 可压缩LuaTable的最大层数
	_MAX_DEPTH = 32
)

func combineType(t, v uint8) uint8 {
	return t | (v << 3)
}

type block struct {
	next   *block
	buffer [_BLOCK_SIZE]byte
}

type writeBlock struct {
	head    *block
	current *block
	len     int // all blocks total length
	ptr     int // current block offset
}

func (wb *writeBlock) init() {
	wb.head = &block{}
	wb.len = 0
	wb.current = wb.head
	wb.ptr = 0
}

func (wb *writeBlock) free() {
	wb.head = nil
	wb.current = nil
	wb.len = 0
	wb.ptr = 0
}

func (wb *writeBlock) mergeBlocks() []byte {
	sz := wb.len
	blk := wb.head
	mergeBuffer := make([]byte, sz)
	ptr := 0
	for sz > 0 {
		if sz >= _BLOCK_SIZE {
			copy(mergeBuffer[ptr:], blk.buffer[:])
			ptr += _BLOCK_SIZE
			sz -= _BLOCK_SIZE
			blk = blk.next
		} else {
			copy(mergeBuffer[ptr:], blk.buffer[:sz])
			break
		}
	}
	return mergeBuffer
}

func (wb *writeBlock) push(buf []byte) {
	sz := len(buf)
	if wb.ptr == _BLOCK_SIZE {
		wb.current.next = &block{}
		wb.current = wb.current.next
		wb.ptr = 0
	}
_again:
	if wb.ptr <= _BLOCK_SIZE-sz {
		copy(wb.current.buffer[wb.ptr:], buf)
		wb.ptr += sz
		wb.len += sz
	} else {
		forward := _BLOCK_SIZE - wb.ptr
		copy(wb.current.buffer[wb.ptr:], buf[:forward])
		buf = buf[forward:]
		wb.len += forward
		sz -= forward
		// new block
		wb.current.next = &block{}
		wb.current = wb.current.next
		wb.ptr = 0
		goto _again
	}
}

func (wb *writeBlock) pushI8(v uint8) {
	wb.push([]byte{v})
}

func (wb *writeBlock) pushI16(v uint16) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	wb.push(buf[:])
}

func (wb *writeBlock) pushI32(v uint32) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], v)
	wb.push(buf[:])
}

func (wb *writeBlock) pushI64(v uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	wb.push(buf[:])
}

func (wb *writeBlock) packNil() {
	wb.pushI8(_TYPE_NIL)
}

func (wb *writeBlock) packBoolean(v bool) {
	b := uint8(0)
	if v {
		b = 1
	}
	wb.pushI8(combineType(_TYPE_BOOLEAN, b))
}

func (wb *writeBlock) packInteger(v int64) {
	if v == 0 {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_ZERO))
	} else if v != int64(int32(v)) {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_QWORD))
		wb.pushI64(uint64(v))
	} else if v < 0 {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_DWORD))
		wb.pushI32(uint32(v))
	} else if v < 0x100 {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_BYTE))
		wb.pushI8(uint8(v))
	} else if v < 0x10000 {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_WORD))
		wb.pushI16(uint16(v))
	} else {
		wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_DWORD))
		wb.pushI32(uint32(v))
	}
}

func (wb *writeBlock) packReal(v float64) {
	wb.pushI8(combineType(_TYPE_NUMBER, _TYPE_NUMBER_REAL))
	wb.pushI64(math.Float64bits(v))
}

func (wb *writeBlock) packString(v string) {
	sz := len(v)
	if sz < _MAX_COOKIE {
		wb.pushI8(combineType(_TYPE_SHORT_STRING, uint8(sz)))
		wb.push([]byte(v))
	} else {
		if sz < 0x10000 {
			wb.pushI8(combineType(_TYPE_LONG_STRING, 2))
			wb.pushI16(uint16(sz))
		} else {
			wb.pushI8(combineType(_TYPE_LONG_STRING, 4))
			wb.pushI32(uint32(sz))
		}
		wb.push([]byte(v))
	}
}

func (wb *writeBlock) packTable(v *Table, depth int) {
	// 填充数组段长度
	arraySize := len(v.Array)
	if arraySize >= _MAX_COOKIE-1 {
		wb.pushI8(combineType(_TYPE_TABLE, _MAX_COOKIE-1))
		wb.packInteger(int64(arraySize))
	} else {
		wb.pushI8(combineType(_TYPE_TABLE, uint8(arraySize)))
	}
	// 压缩数组段
	for i := 0; i < len(v.Array); i++ {
		wb.packOne(v.Array[i], depth)
	}
	// 压缩哈希段
	for k, v := range v.Hashmap {
		wb.packOne(k, depth)
		wb.packOne(v, depth)
	}
	// 填充结束标记
	wb.packNil()
}

func (wb *writeBlock) packOne(v interface{}, depth int) {
	if depth > _MAX_DEPTH {
		wb.free()
		panic("serialize cant't pack too depth table")
	}
	switch data := v.(type) {
	case nil:
		wb.packNil()
	// 自动转换所有整形 -> int64(这一坨暂时没找到优雅的解决方案...)
	case int:
		wb.packInteger(int64(data))
	case uint:
		wb.packInteger(int64(data))
	case int8:
		wb.packInteger(int64(data))
	case uint8:
		wb.packInteger(int64(data))
	case int16:
		wb.packInteger(int64(data))
	case uint16:
		wb.packInteger(int64(data))
	case int32:
		wb.packInteger(int64(data))
	case uint32:
		wb.packInteger(int64(data))
	case int64:
		wb.packInteger(data)
	case uint64:
		wb.packInteger(int64(data))

	case float64:
		wb.packReal(data)
	case bool:
		wb.packBoolean(data)
	case string:
		wb.packString(data)
	case []byte:
		wb.packString(*(*string)(unsafe.Pointer(&data)))
	case *Table:
		wb.packTable(data, depth+1)
	default:
		wb.free()
		panic(fmt.Sprintf("pack unknown type: %T, val: %v", v, v))
	}
}

type readBlock struct {
	buffer []byte
	outs   []interface{}
}

func (rb *readBlock) read(sz int) []byte {
	if len(rb.buffer) < sz {
		return nil
	}
	drain := rb.buffer[:sz]
	rb.buffer = rb.buffer[sz:]
	return drain
}

func (rb *readBlock) popI8() (uint8, bool) {
	drain := rb.read(1)
	if drain == nil {
		return 0, false
	}
	return drain[0], true
}

func (rb *readBlock) popI16() (uint16, bool) {
	drain := rb.read(2)
	if drain == nil {
		return 0, false
	}
	return binary.BigEndian.Uint16(drain), true
}

func (rb *readBlock) popI32() (uint32, bool) {
	drain := rb.read(4)
	if drain == nil {
		return 0, false
	}
	return binary.BigEndian.Uint32(drain), true
}

func (rb *readBlock) popI64() (uint64, bool) {
	drain := rb.read(8)
	if drain == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(drain), true
}

func (rb *readBlock) getReal() float64 {
	n, ok := rb.popI64()
	if !ok {
		rb.invalidStream()
	}
	return math.Float64frombits(n)
}

func (rb *readBlock) getInteger(cookie uint8) int64 {
	switch cookie {
	case _TYPE_NUMBER_ZERO:
		return 0
	case _TYPE_NUMBER_BYTE:
		n, ok := rb.popI8()
		if !ok {
			rb.invalidStream()
		}
		return int64(n)
	case _TYPE_NUMBER_WORD:
		n, ok := rb.popI16()
		if !ok {
			rb.invalidStream()
		}
		return int64(n)
	case _TYPE_NUMBER_DWORD:
		n, ok := rb.popI32()
		if !ok {
			rb.invalidStream()
		}
		return int64(int32(n))
	case _TYPE_NUMBER_QWORD:
		n, ok := rb.popI64()
		if !ok {
			rb.invalidStream()
		}
		return int64(n)
	default:
		rb.invalidStream()
		return 0
	}
}

func (rb *readBlock) getString(cookie uint32) string {
	drain := rb.read(int(cookie))
	if drain == nil {
		rb.invalidStream()
	}
	return string(drain)
}

func (rb *readBlock) unpackOne() {
	_type, ok := rb.popI8()
	if !ok {
		rb.invalidStream()
	}
	rb.pushValue(_type&0x7, _type>>3)
}

func (rb *readBlock) unpackTable(arraySize int) {
	if arraySize == _MAX_COOKIE-1 {
		_type, ok := rb.popI8()
		if !ok {
			rb.invalidStream()
		}
		cookie := _type >> 3
		if (_type&7) != _TYPE_NUMBER || cookie == _TYPE_NUMBER_REAL {
			rb.invalidStream()
		}
		arraySize = int(rb.getInteger(cookie))
	}
	tab := &Table{
		Array:   make([]interface{}, 0, arraySize),
		Hashmap: make(map[interface{}]interface{}),
	}
	rb.outs = append(rb.outs, tab)
	// 解压缩数组段
	for i := 0; i < arraySize; i++ {
		rb.unpackOne()
		// 相当于lua_rawseti(L, -2, i)
		idx := len(rb.outs) - 1
		back := rb.outs[idx]
		rb.outs = append(rb.outs[:idx], rb.outs[idx+1:]...)
		tab.Array = append(tab.Array, back)
	}
	// 解压缩哈希段
	for {
		rb.unpackOne()
		idx := len(rb.outs) - 1
		key := rb.outs[idx]
		// 读取到luaTable的结束标记nil
		if key == nil {
			rb.outs = append(rb.outs[:idx], rb.outs[idx+1:]...)
			return
		}
		rb.unpackOne()
		idx = len(rb.outs) - 1
		val := rb.outs[idx]
		// 相当于lua_rawset(L, -3)
		tab.Hashmap[key] = val
		// 移除key,val
		rb.outs = append(rb.outs[:idx-1], rb.outs[idx+1:]...)
	}
}

func (rb *readBlock) pushValue(_type, cookie uint8) {
	switch _type {
	case _TYPE_NIL:
		rb.outs = append(rb.outs, nil)
	case _TYPE_BOOLEAN:
		rb.outs = append(rb.outs, cookie != 0)
	case _TYPE_NUMBER:
		if cookie == _TYPE_NUMBER_REAL {
			rb.outs = append(rb.outs, rb.getReal())
		} else {
			rb.outs = append(rb.outs, rb.getInteger(cookie))
		}
	case _TYPE_SHORT_STRING:
		rb.outs = append(rb.outs, rb.getString(uint32(cookie)))
	case _TYPE_LONG_STRING:
		if cookie == 2 {
			n, ok := rb.popI16()
			if !ok {
				rb.invalidStream()
			}
			rb.outs = append(rb.outs, rb.getString(uint32(n)))
		} else {
			if cookie != 4 {
				rb.invalidStream()
			}
			n, ok := rb.popI32()
			if !ok {
				rb.invalidStream()
			}
			rb.outs = append(rb.outs, rb.getString(n))
		}
	case _TYPE_TABLE:
		rb.unpackTable(int(cookie))
	default:
		rb.invalidStream()
	}
}

func (rb *readBlock) invalidStream() {
	_, _, line, _ := runtime.Caller(1)
	panic(fmt.Sprintf("Invalid serialize stream %d (line:%d) outs %v", len(rb.buffer), line, rb.outs))
}

func SeriPack(args ...interface{}) []byte {
	wb := &writeBlock{}
	wb.init()
	// lua/seri.c: pack_from
	for _, v := range args {
		wb.packOne(v, 0)
	}
	buffer := wb.mergeBlocks()
	wb.free()
	return buffer
}

func SeriUnpack(buffer []byte) []interface{} {
	rb := &readBlock{
		buffer: buffer,
		// 大部分情况rpc传参不会超过7个
		outs: make([]interface{}, 0, 8),
	}
	for {
		_type, ok := rb.popI8()
		if !ok {
			break
		}
		rb.pushValue(_type&0x7, _type>>3)
	}
	return rb.outs
}
