package serialization

import (
	"encoding/json"
	"io"
)

var _ TextSerializer = (*JsonSerializer)(nil)

type JsonSerializer struct {
}

func (s *JsonSerializer) Serialize(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (s *JsonSerializer) Deserialize(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (s *JsonSerializer) TextSerializer() {}
