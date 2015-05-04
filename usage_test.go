package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestMaxKeys(t *testing.T) {
	c := NewMaxKeysCache(5)

	//Oldest Key should expire
	c.Put("a", "a")
	c.Put("b", "b")
	c.Put("c", "c")
	c.Put("e", "e")
	c.Put("f", "f")
	c.Put("g", "g")

	//At this point one should be evicted... hopefully a
	_, err := c.GetIfPresent("a")
	if err != ErrNotPresent {
		t.Error("Should have evicted a")
	}
}

func TestExpiresAfterWrite(t *testing.T) {
	c := NewExpiresAfterWriteCache(time.Millisecond * 5)

	c.Put("a", "a")

	//At this point no one should be evicted
	v, err := c.GetIfPresent("a")
	if v != "a" {
		t.Error("Should not have evicted a")
		return
	}

	time.Sleep(time.Millisecond * 3)
	fmt.Println(c.GetIfPresent("a"))
	time.Sleep(time.Millisecond * 9)

	_, err = c.GetIfPresent("a")
	if err != ErrNotPresent {
		t.Error("Should have evicted a")
	}
}

func TestExpiresAfterAccess(t *testing.T) {
	c := NewExpiresAfterAccessCache(time.Millisecond * 5)

	c.Put("a", "a")

	//At this point no one should be evicted
	v, err := c.GetIfPresent("a")
	if v != "a" {
		t.Error("Should not have evicted a")
		return
	}

	time.Sleep(time.Millisecond * 3)
	v, err = c.GetIfPresent("a")
	if v != "a" {
		t.Error("Should not have evicted a")
		return
	}
	time.Sleep(time.Millisecond * 3)
	v, err = c.GetIfPresent("a")
	if v != "a" {
		t.Error("Should not have evicted a")
		return
	}
	time.Sleep(time.Millisecond * 3)
	v, err = c.GetIfPresent("a")
	if v != "a" {
		t.Error("Should not have evicted a")
		return
	}

	time.Sleep(time.Millisecond * 10)
	_, err = c.GetIfPresent("a")
	if err != ErrNotPresent {
		t.Error("Should have evicted a")
	}
}
