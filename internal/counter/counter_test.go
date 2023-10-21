package counter

import (
	"testing"
	"time"
)

func TestCount(t *testing.T) {
	Begin()
	go func() {
		defer End()
		time.Sleep(time.Millisecond * 100)
	}()
	c := count()
	if c != 1 {
		t.Errorf("count: expected %d, actual %d", 1, c)
	}
	time.Sleep(time.Millisecond * 100)
	c = count()
	if c != 0 {
		t.Errorf("count: expected %d, actual %d", 0, c)
	}
}

func TestWait(t *testing.T) {
	Begin()
	go func() {
		defer End()
		time.Sleep(time.Millisecond * 100)
	}()
	<-wait()
	c := count()
	if c != 0 {
		t.Errorf("count: expected %d, actual %d", 0, c)
	}
}
