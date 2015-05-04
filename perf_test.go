package cache

import (
	"strconv"
	"testing"
	"time"
)

func fetchFunc(key string) (interface{}, error) {
	time.Sleep(time.Millisecond)
	return 0, nil
}

func BenchmarkNoCache1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 10)
		fetchFunc(key)
	}
}

func BenchmarkTimeBasedCache1000(b *testing.B) {
	c := NewExpiresAfterWriteCache(time.Minute)
	for i := 0; i < b.N; i++ {
		key := strconv.Itoa(i % 10)
		c.GetWithValueLoader(key, fetchFunc)
	}
}
