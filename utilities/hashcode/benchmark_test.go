package hashcode

import (
	"math/rand"
	"testing"
	"time"
)

func BenchmarkGetHashcode(b *testing.B) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	datas := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		data := make([]byte, 1024)
		for j := 0; j < len(data); j++ {
			data[j] = byte(r.Int31n(256))
		}
		datas[i] = data
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetHashCode(datas[i])
	}
}
