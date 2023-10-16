package serialization

import (
	"bytes"
	"encoding/gob"
)

var _ BinarySerializer = (*GobSerializer)(nil)

type GobSerializer struct {
}

func (s *GobSerializer) Serialize(v any) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	e := gob.NewEncoder(buf)
	err := e.Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *GobSerializer) Deserialize(data []byte, v any) error {
	d := gob.NewDecoder(bytes.NewReader(data))
	return d.Decode(v)
}

func (s *GobSerializer) BinarySerializer() {}
