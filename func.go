package template

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

var (
	zeroValue = reflect.Value{}
	errorType = reflect.TypeOf((*error)(nil)).Elem()
)

func Get(p any, keys ...any) (value reflect.Value, err error) {
	value = reflect.ValueOf(p)
	for _, key := range keys {
		kv := reflect.ValueOf(key)
		if value, err = index(value, kv); err == nil {
			continue
		}

		if name, ok := key.(string); ok {
			if value, err = property(value, name); err == nil {
				continue
			}

			fnNames := possibleFnNames(name)
			for _, fnName := range fnNames {
				if fn, err := method(value, fnName); err == nil {
					if value, err = call(fn); err == nil {
						continue
					}
				}
			}
		}
	}

	return
}

func add(x, y reflect.Value) (reflect.Value, error) {
	return calc(x, y, "+")
}

func sub(x, y reflect.Value) (reflect.Value, error) {
	return calc(x, y, "-")
}

func multiple(x, y reflect.Value) (reflect.Value, error) {
	return calc(x, y, "*")
}

func divide(x, y reflect.Value) (reflect.Value, error) {
	return calc(x, y, "/")
}

func eq(x, y reflect.Value) (reflect.Value, error) {
	x = uncoverInterface(x)
	y = uncoverInterface(y)
	if !x.Type().Comparable() || !y.Type().Comparable() {
		return reflect.ValueOf(false), errors.Errorf("con't compare type %s and %b", x.Type(), y.Type())
	}
	if x.Type().Kind() != y.Type().Kind() {
		if isNumber(x.Kind()) && isNumber(y.Kind()) {
			if z, err := sub(x, y); err == nil {
				var r bool
				switch {
				case isFloat(z.Kind()):
					r = z.Float() == 0
				case isInt(z.Kind()):
					r = z.Int() == 0
				case isUint(z.Kind()):
					r = z.Uint() == 0
				}

				return reflect.ValueOf(r), nil
			}
		}
	} else {
		var r bool
		if isInt(x.Kind()) {
			r = x.Int() == y.Int()
		}
		if isUint(x.Kind()) {
			r = x.Uint() == y.Uint()
		}
		if isFloat(x.Kind()) {
			r = x.Float() == y.Float()
		}
		if x.Kind() == reflect.String {
			r = x.String() == y.String()
		}
		if x.Kind() == reflect.Bool {
			r = x.Bool() == y.Bool()
		}
		if x.CanInterface() {
			r = x.Interface() == y.Interface()
		}

		return reflect.ValueOf(r), nil
	}

	return reflect.ValueOf(false), nil
}

func neq(x, y reflect.Value) (reflect.Value, error) {
	if r, err := eq(x, y); err != nil {
		return r, err
	} else {
		return reflect.ValueOf(!r.Bool()), nil
	}
}

func calc(x, y reflect.Value, op string) (reflect.Value, error) {
	x = uncoverInterface(x)
	y = uncoverInterface(y)

	if isNumber(x.Kind()) && isNumber(y.Kind()) {
		var z, a, b any
		if isFloat(x.Kind()) && y.CanConvert(x.Type()) {
			y = y.Convert(x.Type())
			a, b = x.Float(), y.Float()
		} else if isFloat(y.Kind()) && x.CanConvert(y.Type()) {
			x = x.Convert(y.Type())
			a, b = x.Float(), y.Float()
		} else if isInt(x.Kind()) && y.CanConvert(x.Type()) {
			y = y.Convert(x.Type())
			a, b = x.Int(), y.Int()
		} else if isInt(y.Kind()) && x.CanConvert(y.Type()) {
			x = x.Convert(y.Type())
			a, b = x.Int(), y.Int()
		} else if isUint(x.Kind()) && isUint(y.Kind()) {
			a, b = x.Uint(), y.Uint()
		}

		if a != nil && b != nil {
			switch ai := a.(type) {
			case int64:
				switch op {
				case "+":
					z = ai + b.(int64)
				case "-":
					z = ai - b.(int64)
				case "*":
					z = ai * b.(int64)
				case "/":
					z = ai / b.(int64)
				default:
					return zeroValue, errors.Errorf("unsupported calculation %s", op)
				}

			case uint64:
				switch op {
				case "+":
					z = ai + b.(uint64)
				case "-":
					z = ai - b.(uint64)
				case "*":
					z = ai * b.(uint64)
				case "/":
					z = ai / b.(uint64)
				default:
					return zeroValue, errors.Errorf("unsupported calculation %s", op)
				}
			case float64:
				switch op {
				case "+":
					z = ai + b.(float64)
				case "-":
					z = ai - b.(float64)
				case "*":
					z = ai * b.(float64)
				case "/":
					z = ai / b.(float64)
				default:
					return zeroValue, errors.Errorf("unsupported calculation %s", op)
				}
			}
			if z != nil {
				return reflect.ValueOf(z), nil
			}
		}

		if z != nil {
			return reflect.ValueOf(z), nil
		}
	}

	return zeroValue, errors.Errorf("con't add type %s and type %s", x.Type(), y.Type())
}

func greater(x, y reflect.Value) (reflect.Value, error) {
	x = uncoverInterface(x)
	y = uncoverInterface(y)
	if x.Type().Kind() != y.Type().Kind() {
		if isNumber(x.Kind()) && isNumber(y.Kind()) {
			if z, err := sub(x, y); err == nil {
				var r bool
				switch {
				case isFloat(z.Kind()):
					r = z.Float() > 0
				case isInt(z.Kind()):
					r = z.Int() > 0
				case isUint(z.Kind()):
					r = z.Uint() > 0
				}

				return reflect.ValueOf(r), nil
			}
		}

		return reflect.ValueOf(false), errors.Errorf("con't compare type %s and %s", x.Type(), y.Type())
	} else {
		var r bool
		if isInt(x.Kind()) {
			r = x.Int() > y.Int()
		} else if isUint(x.Kind()) {
			r = x.Uint() > y.Uint()
		} else if isFloat(x.Kind()) {
			r = x.Float() > y.Float()
		} else if x.Kind() == reflect.String {
			r = x.String() > y.String()
		} else {
			return reflect.ValueOf(false), errors.Errorf("con't compare type %s", x.Type())
		}

		return reflect.ValueOf(r), nil
	}
}

func greaterOrEqual(x, y reflect.Value) (reflect.Value, error) {

	if r, err := eq(x, y); r.Bool() || err != nil {
		return r, err
	}

	return greater(x, y)
}

func index(value reflect.Value, index reflect.Value) (reflect.Value, error) {
	value, isNil := uncoverReference(value)
	if !value.IsValid() || isNil {
		return zeroValue, errors.New("index of nil value")
	}

	index = uncoverInterface(index)
	if !index.IsValid() {
		return zeroValue, errors.New("nil index")
	}

	switch value.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		cap := value.Len()
		x, err := prepareValueType(index, reflect.TypeOf(int(0)))
		if err != nil {
			return zeroValue, errors.Errorf("con't use type %s as index", index.Type())
		}
		if x.Int() < 0 || int(x.Int()) > cap {
			return zeroValue, errors.Errorf("out of boundary, got %d", x)
		}

		return value.Index(int(x.Int())), nil

	case reflect.Map:
		kType := value.Type().Key()
		x, err := prepareValueType(index, kType)
		if err != nil {
			return zeroValue, err
		}
		var item reflect.Value
		if item = value.MapIndex(x); !value.IsValid() {
			return zeroValue, errors.Errorf("index %s don't exist in map %s", x, value)
		}

		return item, nil

	default:
		return zeroValue, errors.Errorf("can't index item of type %s", value.Type())

	}
}

func property(value reflect.Value, name string) (reflect.Value, error) {
	if !goodName(name) {
		return zeroValue, errors.Errorf("%s is not a property's name", name)
	}

	value = uncoverInterface(value)

	if value.IsNil() {
		return zeroValue, errors.Errorf("can't get property from nil value")
	}

	switch value.Kind() {
	case reflect.Struct:
		field, exist := value.Type().FieldByName(name)
		if !exist {
			return zeroValue, errors.Errorf("property named %s don't exist in type %s", name, value.Type())
		}

		if !field.IsExported() {
			return zeroValue, errors.Errorf("property named %s isn't exported in type %s", name, value.Type())
		}

		return value.FieldByName(name), nil
	case reflect.Pointer:
		value, isNil := uncoverReference(value)
		if isNil {
			return zeroValue, errors.Errorf("can't get property from nil value")
		}

		return property(value, name)

	default:
		return zeroValue, errors.Errorf("can't get property from non-struct type %s", value.Type())

	}
}

func method(value reflect.Value, name string) (reflect.Value, error) {
	if !goodName(name) {
		return zeroValue, errors.Errorf("%s is not a property's name", name)
	}

	value = uncoverInterface(value)

	if value.IsNil() {
		return zeroValue, errors.Errorf("can't get method from nil value")
	}

	method, exist := value.Type().MethodByName(name)
	if !exist {
		return zeroValue, errors.Errorf("method named %s isn't exist in type %s", name, value.Type())
	}

	if !method.IsExported() {
		return zeroValue, errors.Errorf("method named %s isn't exported in type %s", name, value.Type())
	}

	return value.MethodByName(name), nil
}

func call(fn reflect.Value, args ...reflect.Value) (reflect.Value, error) {
	fn = uncoverInterface(fn)
	if fn.IsValid() {
		return zeroValue, errors.New("call on nil")
	}
	typ := fn.Type()

	if !goodFunc(typ) {
		return reflect.Value{}, errors.Errorf("func return %d values; should be 1 or 2", typ.NumOut())
	}

	if typ.Kind() != reflect.Func {
		return zeroValue, errors.Errorf("call on non-func type %s", fn.Type())
	}

	numIn := typ.NumIn()
	var vType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return zeroValue, errors.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}

		vType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return reflect.Value{}, errors.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}

	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		arg = uncoverInterface(arg)
		// Compute the expected type. Clumsy because of variadics.
		argType := vType
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		}

		var err error
		if argv[i], err = prepareValueType(arg, argType); err != nil {
			return zeroValue, errors.Errorf("call func err, arg %d: %w", i, err)
		}
	}

	return invoke(fn, argv)
}

func invoke(fn reflect.Value, argv []reflect.Value) (reflect.Value, error) {
	out := fn.Call(argv)

	if len(out) == 2 && !out[1].IsNil() {
		return out[0], out[1].Interface().(error)
	}

	return out[0], nil
}

func prepareValueType(value reflect.Value, typ reflect.Type) (reflect.Value, error) {
	value = uncoverInterface(value)
	if !value.IsValid() {
		return zeroValue, errors.Errorf("nil value, should be type %s", typ)
	}

	if value.Type().AssignableTo(typ) {
		return value, nil
	}

	if isInteger(value.Kind()) && isInteger(typ.Kind()) && value.CanConvert(typ) {
		return value.Convert(typ), nil
	}

	return zeroValue, errors.Errorf("value has type %s; should be %s", value.Type(), typ)
}

func isNumber(kind reflect.Kind) bool {
	return isInteger(kind) || isFloat(kind)
}

func isInteger(kind reflect.Kind) bool {
	return isInt(kind) || isUint(kind)
}

func isInt(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	}
	return false
}

func isUint(kind reflect.Kind) bool {
	switch kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}

func isFloat(kind reflect.Kind) bool {
	switch kind {
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func uncoverReference(value reflect.Value) (reflect.Value, bool) {
	for ; value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer; value = value.Elem() {
		if value.IsNil() {
			return zeroValue, true
		}
	}

	return value, false
}

func uncoverInterface(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return zeroValue
	}
	if value.Kind() != reflect.Interface {
		return value
	}

	return value.Elem()
}

func goodName(name string) bool {
	if name == "" {
		return false
	}

	for i, r := range name {
		switch {
		case r == '_':
		case i == 0 && !unicode.IsLetter(r):
			return false
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			return false
		}
	}

	return true
}

func goodFunc(typ reflect.Type) bool {
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}

	return false
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
