package hashcode

import (
	"github.com/berkaroad/squat/utilities/hash"
)

func GetHashCode(data []byte) uint64 {
	s := hash.New64()
	s.Write(data)
	return s.Sum64()
}
