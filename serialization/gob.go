package serialization

import (
	"encoding/gob"
	"io"
)

var _ BinarySerializer = (*GobSerializer)(nil)

type GobSerializer struct {
}

func (s *GobSerializer) Serialize(w io.Writer, v any) error {
	return gob.NewEncoder(w).Encode(v)
}

func (s *GobSerializer) Deserialize(r io.Reader, v any) error {
	return gob.NewDecoder(r).Decode(v)
}

func (s *GobSerializer) BinarySerializer() {}
