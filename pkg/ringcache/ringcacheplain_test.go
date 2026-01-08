package ringcache

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRingCachePlain(t *testing.T) {
	rc := NewRingCache[int](10)
	for i := 0; i < 11; i++ {
		rc.Put(i)
	}

	res := make([]int, 0, 11)
	// res := make([]int, 11)  // error
	exp := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for {
		val, ok := rc.Get()
		if !ok {
			break
		}
		fmt.Println(val)
		res = append(res, val)
	}
	if !reflect.DeepEqual(res, exp) {
		t.Errorf("got %v, want %v", res, exp)
	}
}
