package spsc

func TestSPSCRingCache(t *testing.T) {
	rc := NewRingCache(10)
	for i := 0; i < 11; i++ {
		rc.Put(i)
	}

	res := make([]int, 0, 11)
	for {
		val, ok := rc.Get()
		if ok {
			res = append(res, val)
		}
	}
	if !reflect.DeepEqual(res, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Errorf("got %v, want %v", res, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	}
}
