package hashcode

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGetHashCode(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))
	t.Run("same data should returns same hash code", func(t *testing.T) {
		data := make([]byte, 1000)
		for i := 0; i < len(data); i++ {
			data[i] = byte(r.Int31n(256))
		}
		code1 := GetHashCode(data)
		code2 := GetHashCode(data)

		if code1 != code2 {
			t.Errorf("same val should returns same hashcode")
		}
	})

	t.Run("dispersion should be well", func(t *testing.T) {
		testCount := 100_000
		codes := make(map[uint64]int)
		for i := 0; i < testCount; i++ {
			data := make([]byte, 1024)
			for j := 0; j < len(data); j++ {
				data[j] = byte(r.Int31n(256))
			}
			code := GetHashCode(data)
			codes[code] = codes[code] + 1
		}

		diff := testCount - len(codes)
		if math.Abs(float64(diff)) > 0 {
			t.Errorf("dispersion should be well")
		}
	})
}
