package serialization

var defaultSerializer Serializer = &JsonSerializer{}

func Default() Serializer {
	return defaultSerializer
}

type Serializable interface {
	TypeName() string
}

type Serializer interface {
	Serialize(v any) ([]byte, error)
	Deserialize(data []byte, v any) error
}

func Serialize(serializer Serializer, v Serializable) ([]byte, error) {
	if serializer == nil {
		serializer = defaultSerializer
	}
	return serializer.Serialize(v)
}

func Deserialize(serializer Serializer, typeName string, data []byte) (any, error) {
	if serializer == nil {
		serializer = defaultSerializer
	}
	v, err := NewFromTypeName(typeName)
	if err != nil {
		return nil, err
	}
	err = serializer.Deserialize(data, &v)
	return v, err
}
