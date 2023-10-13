package hashcode

func GetHashCode(data []byte) uint64 {
	var h uint64
	for i := 0; i < len(data); i++ {
		h = 31*h + uint64(data[i])
	}
	return h
}
