package seri

import "fmt"

type Table struct {
	// 对应lua table 数组段
	Array []interface{}
	// 对应lua table 哈希段
	Hashmap map[interface{}]interface{}
}

func (t *Table) String() string {
	return fmt.Sprintf(`
Table{
	Arrary: %+v
	Hashmap: %+v
}`, t.Array, t.Hashmap)
}
