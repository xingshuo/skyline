package seri

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerialize(t *testing.T) {
	buffer := SeriPack(500, "abcd", false, []byte("lakefu"), nil, &Table{
		Array: []interface{}{
			"school", nil, uint16(8000), &Table{
				Hashmap: map[interface{}]interface{}{
					"key":       int64(90),
					int64(7000): "good",
				},
			},
		},
		Hashmap: map[interface{}]interface{}{
			"tencent alibaba huawei bytedance pinduoduo": "996",
			9.9: true,
		},
	})
	args := SeriUnpack(buffer)
	assert.Equal(t, args[0], int64(500))
	assert.Equal(t, args[1], "abcd")
	assert.Equal(t, args[2], false)
	assert.Equal(t, args[3], "lakefu")
	assert.Nil(t, args[4])
	assert.Nil(t, args[5].(*Table).Array[1])
	assert.Equal(t, args[5].(*Table).Array[2], int64(8000))
	assert.Equal(t, args[5].(*Table).Array[3].(*Table).Hashmap[int64(7000)], "good")
	assert.Equal(t, args[5].(*Table).Hashmap[9.9], true)
}
