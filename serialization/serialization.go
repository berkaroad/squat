package serialization

import (
	"bytes"
	"encoding/base64"
	"io"
)

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
	Serialize(w io.Writer, v any) error
	Deserialize(r io.Reader, v any) error
}

type BinarySerializer interface {
	Serializer
	BinarySerializer()
}

type TextSerializer interface {
	Serializer
	TextSerializer()
}

func Serialize(serializer Serializer, r io.Writer, v any) error {
	if serializer == nil {
		serializer = defaultTextSerializer
	}

	err := serializer.Serialize(r, v)
	if err != nil {
		return err
	}
	return nil
}

func Deserialize(serializer Serializer, typeName string, r io.Reader) (any, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	vInterface, err := NewFromTypeName(typeName)
	if err != nil {
		return nil, err
	}
	v := vInterface.Interface()
	err = serializer.Deserialize(r, v)
	return v, err
}

func SerializeToText(serializer Serializer, v any) (string, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	err := serializer.Serialize(buf, v)
	if err != nil {
		return "", err
	}
	var data string
	if _, ok := serializer.(TextSerializer); ok {
		data = buf.String()
	} else {
		data = base64.StdEncoding.EncodeToString(buf.Bytes())
	}
	return data, nil
}

func DeserializeFromText(serializer Serializer, typeName string, text string) (any, error) {
	if serializer == nil {
		serializer = defaultTextSerializer
	}
	vInterface, err := NewFromTypeName(typeName)
	if err != nil {
		return nil, err
	}

	var data []byte
	if _, ok := serializer.(TextSerializer); ok {
		data = []byte(text)
	} else {
		var err error
		if data, err = base64.StdEncoding.DecodeString(text); err != nil {
			return nil, err
		}
	}

	v := vInterface.Interface()
	err = serializer.Deserialize(bytes.NewReader(data), v)
	return v, err
}
