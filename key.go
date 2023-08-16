package cacheDatabase

import (
	"fmt"
	"reflect"
)

func getListKeys(object interface{}) (listKeys []string) {
	objValue := reflect.ValueOf(object)
	method := objValue.MethodByName("ListKeyUnique")

	// Display list method (DEBUG)
	/*
		typeOfP := reflect.TypeOf(object)
		for i := 0; i < typeOfP.NumMethod(); i++ {
			method := typeOfP.Method(i)
			fmt.Println(" ====> Méthode:", method.Name)
		}
	*/

	if method.IsValid() {
		listArgValue := method.Call([]reflect.Value{})

		if len(listArgValue) < 1 {
			return make([]string, 0)
		}
		firstArgValue := listArgValue[0]
		if firstArgValue.Kind() != reflect.Array && firstArgValue.Kind() != reflect.Slice {
			return make([]string, 0)
		}

		length := firstArgValue.Len()
		listKeys = make([]string, length)
		for i := 0; i < length; i++ {
			listKeys[i] = fmt.Sprintf("%v", firstArgValue.Index(i).Interface())
		}
		return listKeys
	}
	return make([]string, 0)
}

func getValueOfKey(object interface{}, fieldname string) string {
	value := reflect.ValueOf(object)
	if value.Kind() == reflect.Pointer {
		return getValueOfKey(value.Elem().Interface(), fieldname)
	}
	field := value.FieldByName(fieldname)

	if field.IsValid() {
		return fmt.Sprintf("%v", field.Interface())
	}
	return ""
}
