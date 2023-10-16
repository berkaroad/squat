package serialization

import (
	"fmt"
	"reflect"
	"sync"
)

var typeMapper sync.Map //map[string]reflect.Type = make(map[string]reflect.Type)

// Map struct type to type's name.
// Thread unsafe, should be invoked at startup.
func Map[T any]() {
	var t T
	typ := reflect.TypeOf(t)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	MapTo[T](typ.Name())
}

// MapTo to map struct type to custom type name.
// Thread unsafe, should be invoked at startup.
func MapTo[T any](typeName string) {
	var t T
	typ := reflect.TypeOf(t)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("typeName '%s' mapping fail: only struct or pointer to struct can be mapped, T '%s/%s'", typeName, reflect.TypeOf(t).PkgPath(), reflect.TypeOf(t).Name()))
	}

	if existsObj, ok := typeMapper.Load(typeName); ok {
		exists := existsObj.(reflect.Type)
		if exists != typ {
			panic(fmt.Sprintf("typeName '%s' mapping fail: has already mapped by '%s/%s'", typeName, exists.PkgPath(), exists.Name()))
		}
	} else {
		typeMapper.Store(typeName, typ)
	}
}

func GetType(typeName string) (reflect.Type, error) {
	valObj, ok := typeMapper.Load(typeName)
	if !ok {
		return nil, fmt.Errorf("typeName '%s' has not registered", typeName)
	}
	val := valObj.(reflect.Type)
	return val, nil
}

func NewFromTypeName(typeName string) (any, error) {
	typ, err := GetType(typeName)
	if err != nil {
		return nil, err
	}
	return reflect.New(typ).Interface(), nil
}
