package serialization

import "encoding/base64"

var defaultTextSerializer TextSerializer = &JsonSerializer{}
var defaultBinarySerializer BinarySerializer = &GobSerializer{}

func DefaultText() TextSerializer {
	return defaultTextSerializer
}

func DefaultBinary() BinarySerializer {
	return defaultBinarySerializer
}

type Serializable interface {
	TypeName() string
}

type Serializer interface {
	Serialize(v any) ([]byte, error)
	Deserialize(data []byte, v any) error
}

type BinarySerializer interface {
	Serializer
	BinarySerializer()
}

type TextSerializer interface {
	Serializer
	TextSerializer()
}

func Serialize(serializer Serializer, v any) ([]byte, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	return serializer.Serialize(v)
}

func SerializeToString(serializer Serializer, v any) (string, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	dataBytes, err := serializer.Serialize(v)
	if err != nil {
		return "", err
	}
	var data string
	if _, ok := serializer.(TextSerializer); ok {
		data = string(dataBytes)
	} else {
		data = base64.StdEncoding.EncodeToString(dataBytes)
	}
	return data, nil
}

func Deserialize(serializer Serializer, typeName string, data []byte) (any, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	v, err := NewFromTypeName(typeName)
	if err != nil {
		return nil, err
	}
	err = serializer.Deserialize(data, v)
	return v, err
}
