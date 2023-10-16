package serialization

import "testing"

type Class1 struct {
	ID string
}

func deserialize(t *testing.T, serializer Serializer) {
	t.Run("serialize and deserialize *struct{} should success", func(t *testing.T) {
		MapTo[*Class1]("Class1")
		data1 := &Class1{ID: "agg1"}
		data1Bytes, err := Serialize(serializer, data1)
		if err != nil {
			t.Error("serialize *struct{} fail")
			return
		}
		dataInterface, err := Deserialize(serializer, "Class1", data1Bytes)
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
		MapTo[Class1]("Class1")
		data1 := Class1{ID: "agg1"}
		data1Bytes, err := Serialize(serializer, data1)
		if err != nil {
			t.Error("serialize struct{} fail")
			return
		}
		dataInterface, err := Deserialize(serializer, "Class1", data1Bytes)
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
