package tests

import (
	"fmt"
	"sync/atomic"
)

var id uint32 = 0

func NewUUID() string {
	return fmt.Sprintf("%d", atomic.AddUint32(&id, 1))
}
