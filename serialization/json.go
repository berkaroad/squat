package serialization

import "encoding/json"

var _ Serializer = (*JsonSerializer)(nil)

type JsonSerializer struct {
}

func (s *JsonSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JsonSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
