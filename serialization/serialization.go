package serialization

var DefaultSerializer Serializer = JsonSerializer{}

type Serializable interface {
	TypeName() string
}

type Serializer interface {
	Serialize(v any) ([]byte, error)
	Deserialize(data []byte, v any) error
}

func Serialize(serializer Serializer, v Serializable) ([]byte, error) {
	if serializer == nil {
		serializer = DefaultSerializer
	}
	return serializer.Serialize(v)
}

func Deserialize(serializer Serializer, typeName string, data []byte) (any, error) {
	if serializer == nil {
		serializer = DefaultSerializer
	}
	v, err := NewFromTypeName(typeName)
	if err != nil {
		return nil, err
	}
	err = serializer.Deserialize(data, &v)
	return v, err
}
