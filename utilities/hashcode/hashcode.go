package hashcode

import (
	"encoding/binary"
	"hash"
)

func GetHashCode(data []byte) uint64 {
	s := New64()
	s.Write(data)
	return s.Sum64()
}

type (
	sum32 uint32
	sum64 uint64
)

const (
	offset32 = 2_147_483_648
	offset64 = 2_305_843_009_213_693_952
	prime32  = 31
	prime64  = 61
)

func New32() hash.Hash32 {
	var s sum32 = offset32
	return &s
}

func New64() hash.Hash64 {
	var s sum64 = offset64
	return &s
}

func (s *sum32) Reset() { *s = offset32 }
func (s *sum64) Reset() { *s = offset64 }

func (s *sum32) Sum32() uint32 { return uint32(*s) }
func (s *sum64) Sum64() uint64 { return uint64(*s) }

func (s *sum32) Write(data []byte) (int, error) {
	hash := *s
	for _, c := range data {
		hash = prime32*hash + sum32(c)
	}
	*s = hash
	return len(data), nil
}

func (s *sum64) Write(data []byte) (int, error) {
	hash := *s
	for _, c := range data {
		hash = prime64*hash + sum64(c)
	}
	*s = hash
	return len(data), nil
}

func (s *sum32) Size() int { return 4 }
func (s *sum64) Size() int { return 8 }

func (s *sum32) BlockSize() int { return 1 }
func (s *sum64) BlockSize() int { return 1 }

func (s *sum32) Sum(in []byte) []byte {
	v := uint32(*s)
	return binary.BigEndian.AppendUint32(in, v)
}

func (s *sum64) Sum(in []byte) []byte {
	v := uint64(*s)
	return binary.BigEndian.AppendUint64(in, v)
}
