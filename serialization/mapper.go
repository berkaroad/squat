package serialization

import (
	"fmt"
	"reflect"
	"sync"
)

var typeMapper sync.Map //map[string]reflect.Type = make(map[string]reflect.Type)

// Map struct type to type's name.
//
// Use T.TypeName() as type's name if T implements interface{TypeName() string}, else use struct type's name.
// Thread unsafe, should be invoked at startup.
func Map[T any]() {
	var t T
	typ := reflect.TypeOf(t)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mapping fail: only struct or pointer to struct can be mapped, T '%s/%s'", reflect.TypeOf(t).PkgPath(), reflect.TypeOf(t).Name()))
	}

	typeName := typ.Name()
	obj := reflect.New(typ).Interface()
	if serializableObj, ok := obj.(Serializable); ok {
		typeName = serializableObj.TypeName()
	}
	if existsObj, loaded := typeMapper.LoadOrStore(typeName, typ); loaded {
		if exists := existsObj.(reflect.Type); exists != typ {
			panic(fmt.Sprintf("typeName '%s' mapping fail: has already mapped by '%s/%s'", typeName, exists.PkgPath(), exists.Name()))
		}
	}
}

// GetType to get struct type that has mapped by type name.
func GetType(typeName string) (reflect.Type, error) {
	valObj, ok := typeMapper.Load(typeName)
	if !ok {
		return nil, fmt.Errorf("typeName '%s' has not registered", typeName)
	}
	val := valObj.(reflect.Type)
	return val, nil
}

// NewFromTypeName to new struct value that has mapped by type name.
func NewFromTypeName(typeName string) (reflect.Value, error) {
	typ, err := GetType(typeName)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.New(typ), nil
}
