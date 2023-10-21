package serialization

import (
	"bytes"
	"testing"
)

var _ Serializable = (*Class1)(nil)

type Class1 struct {
	ID string
}

func (c *Class1) TypeName() string {
	return "Class1"
}

func deserialize(t *testing.T, serializer Serializer) {
	t.Run("serialize and deserialize *struct{} should success", func(t *testing.T) {
		Map[*Class1]()
		Map[*Class1]()
		Map[Class1]()
		Map[Class1]()
		data1 := &Class1{ID: "agg1"}
		buf := bytes.NewBuffer(make([]byte, 0, 64))
		err := Serialize(serializer, buf, data1)
		if err != nil {
			t.Error("serialize *struct{} fail")
			return
		}
		dataInterface, err := Deserialize(serializer, "Class1", bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Error("deserialize to *struct{} fail")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("deserialize to *struct{} fail, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("not equal with last one")
		}
	})

	t.Run("serialize and deserialize struct{} should success", func(t *testing.T) {
		Map[*Class1]()
		Map[*Class1]()
		Map[Class1]()
		Map[Class1]()
		data1 := Class1{ID: "agg1"}
		buf := bytes.NewBuffer(make([]byte, 0, 64))
		err := Serialize(serializer, buf, data1)
		if err != nil {
			t.Error("serialize struct{} fail")
			return
		}
		dataInterface, err := Deserialize(serializer, "Class1", buf)
		if err != nil {
			t.Error("deserialize to struct{} fail")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("deserialize to struct{} fail, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("not equal with last one")
		}
	})
}

func deserializeFromText(t *testing.T, serializer Serializer) {
	t.Run("serialize and deserialize *struct{} should success", func(t *testing.T) {
		Map[*Class1]()
		Map[*Class1]()
		Map[Class1]()
		Map[Class1]()
		data1 := &Class1{ID: "agg1"}
		data1Str, err := SerializeToText(serializer, data1)
		if err != nil {
			t.Error("serialize *struct{} fail")
			return
		}
		dataInterface, err := DeserializeFromText(serializer, "Class1", data1Str)
		if err != nil {
			t.Error("deserialize to *struct{} fail")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("deserialize to *struct{} fail, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("not equal with last one")
		}
	})

	t.Run("serialize and deserialize struct{} should success", func(t *testing.T) {
		Map[*Class1]()
		Map[*Class1]()
		Map[Class1]()
		Map[Class1]()
		data1 := Class1{ID: "agg1"}
		data1Str, err := SerializeToText(serializer, data1)
		if err != nil {
			t.Error("serialize struct{} fail")
			return
		}
		dataInterface, err := DeserializeFromText(serializer, "Class1", data1Str)
		if err != nil {
			t.Error("deserialize to struct{} fail")
			return
		}
		if data, ok := dataInterface.(*Class1); !ok {
			t.Error("deserialize to struct{} fail, couldn't convert to *Class1")
		} else if data.ID != "agg1" {
			t.Error("not equal with last one")
		}
	})
}
