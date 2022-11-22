package converter

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Encode copies user defined structs into terraform shcema.resourceData struct
func Encode(v interface{}, d *schema.ResourceData) error {
	return encode(v, d)
}

func encode(vI interface{}, resourceDataI interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = fmt.Errorf("unknown panic: %+v", x)
			}
		}
	}()
	v := reflect.ValueOf(vI)
	resourceDataV := reflect.ValueOf(resourceDataI)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := v.Type().Field(i).Tag
		fieldName, ok := tag.Lookup("name")
		if !ok {
			continue
		}

		rd := resourceDataV.MethodByName("Get").Call([]reflect.Value{reflect.ValueOf(fieldName)})[0]
		if rd.IsNil() {
			continue
		}
		newDestinationElement := reflect.New(convertType(field.Type()))
		copyEncode(newDestinationElement, field)
		resourceDataV.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(fieldName), reflect.ValueOf(reflect.Indirect(newDestinationElement).Interface())})

	}
	return err
}

func copyEncode(destinationPointer reflect.Value, source reflect.Value) {
	destination := reflect.Indirect(destinationPointer)
	if source.Kind() == reflect.Slice {
		for i := 0; i < source.Len(); i++ {
			newDestinationElement := reflect.New(convertType(source.Index(i).Type()))
			copyEncode(newDestinationElement, source.Index(i))
			destination.Set(reflect.Append(destination, reflect.Indirect(newDestinationElement)))
		}
	} else if source.Kind() == reflect.Struct {
		mp := reflect.MakeMap(convertType(source.Type()))
		for i := 0; i < source.NumField(); i++ {
			tag := source.Type().Field(i).Tag
			fieldName, ok := tag.Lookup("name")
			if !ok {
				continue
			}
			newDestinationElement := reflect.New(convertType(source.Field(i).Type()))
			copyEncode(newDestinationElement, source.Field(i))
			mp.SetMapIndex(reflect.ValueOf(fieldName), reflect.Indirect(newDestinationElement))
		}
		destination.Set(mp)
	} else if source.Kind() == reflect.Map {
		mp := reflect.MakeMap(convertType(source.Type()))
		for _, key := range source.MapKeys() {
			newDestinationElement := reflect.New(convertType(source.MapIndex(key).Type()))
			copyEncode(newDestinationElement, source.MapIndex(key))
			keyString := stringifyKey(key)
			mp.SetMapIndex(keyString, reflect.Indirect(newDestinationElement))
		}
		destination.Set(mp)
	} else {

		switch source.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			destination.SetInt(source.Int())
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			destination.SetInt(int64(source.Uint()))
		case reflect.Float32, reflect.Float64:
			destination.SetFloat(source.Float())
		default:
			destination.Set(source)
		}
	}

}

func stringifyKey(k reflect.Value) reflect.Value {
	var keyString reflect.Value
	switch k.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		keyString = reflect.ValueOf(strconv.FormatInt(k.Int(), 10))
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		keyString = reflect.ValueOf(strconv.FormatUint(k.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		keyString = reflect.ValueOf(strconv.FormatFloat(k.Float(), 'f', -1, 64))
	case reflect.Bool:
		keyString = reflect.ValueOf(strconv.FormatBool(k.Bool()))
	case reflect.String:
		keyString = k
	}
	return keyString
}

func convertType(t reflect.Type) reflect.Type {
	var ret reflect.Type
	switch t.Kind() {
	case reflect.Struct, reflect.Map:
		ret = reflect.TypeOf(map[string]interface{}{})
	case reflect.Slice, reflect.Array:
		ret = reflect.TypeOf([]interface{}{})
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		ret = reflect.TypeOf(int(1))
	case reflect.Float32, reflect.Float64:
		ret = reflect.TypeOf(float32(1))
	default:
		ret = t
	}
	return ret
}
