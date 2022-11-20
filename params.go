package template

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Params struct {
	store map[string]any
}

func (p *Params) Get(key string) any {
	return p.store[key]
}

func (p *Params) Set(key string, val any) {
	p.store[key] = val
}

func GetDot(keys string, p any) (val any, err error) {
	keyArr := strings.Split(keys, ".")
	val = p
	for _, key := range keyArr {
		if val, err = Get(key, val); err != nil {
			return
		}
	}

	return
}

func Get(key string, p any) (any, error) {
	value := reflect.ValueOf(p)

	return getDataFromValue(key, value)
}

func getDataFromValue(key string, value reflect.Value) (any, error) {
	typ := value.Type()
	if typ.Kind() == reflect.Map {
		kValue := reflect.ValueOf(key)

		return value.MapIndex(kValue).Interface(), nil
	} else if typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice {
		index, err := strconv.Atoi(key)
		if err != nil {
			return nil, err
		}
		kValue := value.Index(index)

		return kValue.Interface(), nil
	} else if hasField(ucFirst(key), value) {
		return getDataFromField(ucFirst(key), value), nil
	} else {
		fns := possibleFnNames(key)
		for _, fn := range fns {
			if hasMethod(fn, value) {
				return callFunc(fn, value)
			}
		}
		if hasMethod("Get", value) {
			return callFunc("Get", value, key)
		}
	}

	return nil, errors.Errorf("can't get \"%s\" from %s", key, value.Interface())
}

func hasMethod(fn string, value reflect.Value) bool {
	method := value.MethodByName(fn)

	if method.IsValid() {
		outNum := method.Type().NumOut()
		if outNum != 2 && outNum != 1 {
			panic(fmt.Sprintf("func %s of %s must return 1 or 2 value(s), got %d",
				fn,
				value.Type().Name(),
				outNum,
			))
		}
	}

	return method.IsValid()
}

func callFunc(fn string, value reflect.Value, args ...any) (any, error) {
	a := []reflect.Value{}
	for _, v := range args {
		a = append(a, reflect.ValueOf(v))
	}
	method := value.MethodByName(fn)
	outNum := method.Type().NumOut()
	values := method.Call(a)

	if outNum == 1 {
		return values[0].Interface(), nil
	} else {
		if values[1].IsNil() {
			return values[0].Interface(), nil
		}

		if err, ok := values[1].Interface().(error); ok {
			return nil, err
		}

		panic(fmt.Sprintf("the second out value of %s must be error type, got %s",
			fn,
			values[1].Type().Name(),
		))
	}
}

func hasField(name string, value reflect.Value) bool {
	typ := value.Type()
	if typ.Kind() == reflect.Pointer {
		value = value.Elem()

		return hasField(name, value)
	}

	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("con't get \"%s\" from %s(%s)",
			name,
			value.Interface(),
			value.Type().Name(),
		))
	}

	field, ok := typ.FieldByName(name)

	return ok && field.IsExported()
}

func getDataFromField(name string, value reflect.Value) any {
	typ := value.Type()
	if typ.Kind() == reflect.Pointer {
		value = value.Elem()

		return getDataFromField(name, value)
	}

	field := value.FieldByName(name)

	return field.Interface()
}

func possibleFnNames(word string) []string {
	word = ucFirst(word)
	return []string{
		word,
		fmt.Sprintf("Get%s", word),
		fmt.Sprintf("Has%s", word),
		fmt.Sprintf("Is%s", word),
	}
}

func ucFirst(word string) string {
	c := word[0:1]
	uc := strings.ToUpper(c)

	return strings.Replace(word, c, uc, 1)
}
