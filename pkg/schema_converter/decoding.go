package converter

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Decode(v interface{}, d *schema.ResourceData) error {
	return decode(v, d)
}

// decode extracts data from terraform schema.resourceData struct into user defined struct
func decode(vI interface{}, resourceDataI interface{}) (err error) {
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
	v := reflect.Indirect(reflect.ValueOf(vI))
	resourceDataV := reflect.ValueOf(resourceDataI)
	for i := 0; i < v.NumField(); i++ {
		tag := v.Type().Field(i).Tag
		fieldName, ok := tag.Lookup("name")
		if !ok {
			continue
		}
		fvalue := resourceDataV.MethodByName("Get").Call([]reflect.Value{reflect.ValueOf(fieldName)})[0]
		if fvalue.IsNil() {
			continue
		}
		newDestinationElement := reflect.New(v.Field(i).Type())
		copyDecode(newDestinationElement.Interface(), fvalue.Interface())
		v.Field(i).Set(reflect.Indirect(newDestinationElement))
	}
	return err
}

func copyDecode(destinationI interface{}, sourceI interface{}) {
	destinationV := reflect.Indirect(reflect.ValueOf(destinationI))
	sourceV := reflect.ValueOf(sourceI)
	if destinationV.Kind() == reflect.Slice {
		for i := 0; i < sourceV.Len(); i++ {
			newDestinationElement := reflect.New(destinationV.Type().Elem())
			copyDecode(newDestinationElement.Interface(), sourceV.Index(i).Interface())
			destinationV.Set(reflect.Append(destinationV, reflect.Indirect(newDestinationElement)))
		}
	} else if destinationV.Kind() == reflect.Map {
		keyType := destinationV.Type().Key()
		valType := destinationV.Type().Elem()
		mp := reflect.MakeMap(reflect.MapOf(keyType, valType))
		mpk := sourceV.MapKeys()
		for idx := range mpk {
			newDestinationElement := reflect.New(valType)
			copyDecode(newDestinationElement.Interface(), sourceV.MapIndex(mpk[idx]).Elem().Interface())
			key := reflect.New(keyType)
			getTypeFromString(key, mpk[idx].String())
			mp.SetMapIndex(reflect.Indirect(key), reflect.Indirect(newDestinationElement))
		}
		destinationV.Set(mp)
	} else if destinationV.Kind() == reflect.Struct {
		for i := 0; i < destinationV.NumField(); i++ {
			tag := destinationV.Type().Field(i).Tag
			fieldName, ok := tag.Lookup("name")
			if !ok {
				continue
			}
			if !sourceV.MapIndex(reflect.ValueOf(fieldName)).IsValid() || sourceV.MapIndex(reflect.ValueOf(fieldName)).IsZero() {
				continue
			}
			newDestinationElement := reflect.New(destinationV.Field(i).Type())
			copyDecode(newDestinationElement.Interface(), sourceV.MapIndex(reflect.ValueOf(fieldName)).Interface())
			destinationV.Field(i).Set(reflect.Indirect(newDestinationElement))
		}
	} else {
		switch destinationV.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint8:
			destinationV.SetInt(sourceV.Int())
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			destinationV.SetUint(uint64(sourceV.Int()))
		case reflect.Float32, reflect.Float64:
			destinationV.SetFloat(sourceV.Float())
		case reflect.Bool:
			destinationV.SetBool(sourceV.Bool())
		default:
			destinationV.Set(sourceV)
		}
	}
}

func getTypeFromString(key reflect.Value, s string) {
	v := reflect.Indirect(key)
	switch v.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
		i, _ := strconv.ParseInt(s, 10, 64)
		v.SetInt(i)
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
		i, _ := strconv.ParseUint(s, 10, 64)
		v.SetUint(i)
	case reflect.Float32, reflect.Float64:
		i, _ := strconv.ParseFloat(s, 64)
		v.SetFloat(i)
	case reflect.Bool:
		i, _ := strconv.ParseBool(s)
		v.SetBool(i)
	case reflect.String:
		v.SetString(s)
	}
}
