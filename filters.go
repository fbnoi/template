package template

import (
	"reflect"

	"github.com/pkg/errors"
)

var filters = map[string]reflect.Value{
	"length": reflect.ValueOf(length),
	"P":      reflect.ValueOf(P),
}

func buildInFilters() map[string]reflect.Value {
	return filters
}

func length(i any) (int, error) {
	iValue := reflect.ValueOf(i)
	switch iValue.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return iValue.Len(), nil
	}

	return 0, errors.Errorf("can't get length of type %s", iValue.Type())
}
