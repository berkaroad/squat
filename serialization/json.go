package serialization

import (
	"encoding/json"
	"io"
)

var _ TextSerializer = (*JSONSerializer)(nil)

type JSONSerializer struct {
}

func (s *JSONSerializer) Serialize(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (s *JSONSerializer) Deserialize(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (s *JSONSerializer) TextSerializer() {}
