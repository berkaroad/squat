package serialization

import "encoding/base64"

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
	IsTextSerializer() bool
}

func Serialize(serializer Serializer, v Serializable) ([]byte, error) {
	if serializer == nil {
		serializer = defaultSerializer
	}
	return serializer.Serialize(v)
}

func SerializeToString(serializer Serializer, v Serializable) (string, error) {
	if serializer == nil {
		serializer = defaultSerializer
	}
	dataBytes, err := serializer.Serialize(v)
	if err != nil {
		return "", err
	}
	var data string
	if serializer.IsTextSerializer() {
		data = string(dataBytes)
	} else {
		data = base64.StdEncoding.EncodeToString(dataBytes)
	}
	return data, nil
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

func IsTextSerializer(serializer Serializer) bool {
	if serializer == nil {
		serializer = defaultSerializer
	}
	return serializer.IsTextSerializer()
}
