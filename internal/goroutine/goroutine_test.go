package goroutine

import (
	"context"
	"testing"
	"time"
)

func TestGo(t *testing.T) {
	val := 0
	Go(context.TODO(), func(ctx context.Context) {
		time.Sleep(time.Millisecond * 100)
		val = 654321
	})
	time.Sleep(time.Millisecond * 200)
	if val != 654321 {
		t.Errorf("val: expected %d, actual %d", 654321, val)
	}
}

func TestCount(t *testing.T) {
	Go(context.TODO(), func(ctx context.Context) {
		time.Sleep(time.Millisecond * 100)
	})
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
	Go(context.TODO(), func(ctx context.Context) {
		time.Sleep(time.Millisecond * 100)
	})
	<-wait()
	c := count()
	if c != 0 {
		t.Errorf("count: expected %d, actual %d", 0, c)
	}
}
