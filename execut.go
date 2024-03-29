package template

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func (e *ident) execute(p Params) (reflect.Value, error) {
	return get(p, e.name.value)
}

func (e *basicLit) execute(Params) (reflect.Value, error) {
	vs := e.value.value
	switch e.kind {
	case type_number:
		if i, err := strconv.Atoi(vs); err == nil {
			return reflect.ValueOf(i), nil
		}
		if f, err := strconv.ParseFloat(vs, 64); err == nil {
			return reflect.ValueOf(f), nil
		}

	case type_string:
		vs = trimString(vs)

		return reflect.ValueOf(vs), nil

	case type_bool:
		if i, err := strconv.ParseBool(vs); err == nil {
			return reflect.ValueOf(i), nil
		}
	}

	return zeroValue, newUnexpectedToken(e.value)
}

func (e *listExpr) execute(Params) (reflect.Value, error) {
	return zeroValue, nil
}

func (e *indexExpr) execute(p Params) (reflect.Value, error) {
	x, err := e.x.execute(p)
	if err != nil {
		return zeroValue, err
	}
	vx := x.Interface()
	op := e.op.value
	switch op {
	case ".":
		switch index := e.index.(type) {
		case *ident:
			return get(vx, index.name.value)
		case *callExpr:
			if fn, err := method(x, index.fn.name.value); err != nil {
				return zeroValue, err
			} else {
				argv := []reflect.Value{}
				for _, v := range index.args.list {
					if arg, err := v.execute(p); err == nil {
						argv = append(argv, arg)
					} else {
						return zeroValue, err
					}
				}

				return call(fn, argv...)
			}

		default:
			return zeroValue, newUnexpectedToken(e.op)

		}

	case "[":
		v, err := e.index.execute(p)
		if err != nil {
			return zeroValue, err
		}
		if v.CanInt() {
			return get(vx, x)
		}
		if v.Kind() == reflect.String {
			return get(vx, v.Interface().(string))
		}
		return zeroValue, errors.Errorf("con't convert %s(type of %s) to type string",
			v,
			reflect.TypeOf(v).Name(),
		)

	default:
		return zeroValue, errors.Errorf("unexpected index token %s", op)
	}
}

func (e *callExpr) execute(p Params) (reflect.Value, error) {
	if fn := getFunc(e.fn.name.value); fn != zeroValue {
		argv := []reflect.Value{}
		for _, v := range e.args.list {
			if arg, err := v.execute(p); err == nil {
				argv = append(argv, arg)
			} else {
				return zeroValue, err
			}
		}
		return call(fn, argv...)
	}

	return zeroValue, errors.Errorf("func named %s doesn't exist", e.fn.name.value)
}

func (e *binaryExpr) execute(p Params) (reflect.Value, error) {
	op := e.op.value
	x, err := e.x.execute(p)
	if err != nil {
		return zeroValue, err
	}
	y, err := e.y.execute(p)
	if err != nil {
		return zeroValue, err
	}
	switch op {
	case "+":
		return add(x, y)
	case "-":
		return sub(x, y)
	case "*":
		return multiple(x, y)
	case "/":
		return divide(x, y)
	case ">":
		return greater(x, y)
	case "<":
		return greater(y, x)
	case ">=":
		return greaterOrEqual(x, y)
	case "<=":
		return greaterOrEqual(y, x)
	case "==":
		return eq(x, y)
	case "!=":
		return neq(x, y)
	case "and":
		if t1, err := boolValue(x); err != nil {
			return zeroValue, err
		} else if t2, err := boolValue(y); err != nil {
			return zeroValue, err
		} else {
			return reflect.ValueOf(t1 && t2), nil
		}
	case "or":
		if t1, err := boolValue(x); err != nil {
			return zeroValue, err
		} else if t2, err := boolValue(y); err != nil {
			return zeroValue, err
		} else {
			return reflect.ValueOf(t1 || t2), nil
		}
	}

	return zeroValue, newUnexpectedToken(e.op)
}

func (e *singleExpr) execute(p Params) (reflect.Value, error) {
	x, err := e.x.execute(p)
	if err != nil {
		return zeroValue, err
	}
	switch e.op.value {

	case "not":
		r, err := boolValue(x)
		if err != nil {
			return zeroValue, err
		}

		return reflect.ValueOf(!r), nil

	}

	return zeroValue, newUnexpectedToken(e.op)
}

func (e *pipelineExpr) execute(p Params) (reflect.Value, error) {
	if x, err := e.x.execute(p); err != nil {
		return zeroValue, err
	} else {
		var (
			filter reflect.Value
			argv   = []reflect.Value{x}
		)
		switch y := e.y.(type) {
		case *ident:
			filter = getFilter(y.name.value)
			if filter == zeroValue {
				return zeroValue, errors.Errorf("filter named %s doesn't exist", y.name.value)
			}

		case *callExpr:
			filter = getFilter(y.fn.name.value)
			for _, v := range y.args.list {
				if arg, err := v.execute(p); err == nil {
					argv = append(argv, arg)
				} else {
					return zeroValue, err
				}
			}

		}

		return call(filter, argv...)
	}
}

func (d *textDirect) execute(p Params) (string, error) {
	return d.text.value.value, nil
}

func (d *valueDirect) execute(p Params) (string, error) {
	if v, err := d.tok.execute(p); err != nil {
		return "", err
	} else {
		return strValue(v)
	}
}

func (d *assignDirect) execute(p Params) (string, error) {
	yx, err := d.rh.execute(p)
	if err != nil {
		return "", err
	}
	p[d.lh.name.value] = yx.Interface()

	return "", nil
}

func (d *sectionDirect) execute(p Params) (string, error) {
	sb := &strings.Builder{}
	for _, x := range d.list {
		if str, err := x.execute(p); err != nil {
			return "", err
		} else {
			sb.WriteString(str)
		}
	}

	return sb.String(), nil
}

func (d *ifDirect) execute(p Params) (string, error) {
	if conv, err := d.cond.execute(p); err != nil {
		return "", err
	} else {
		if truth, err := boolValue(conv); err != nil {
			return "", err
		} else if truth {
			return d.body.execute(p)
		} else if d.el != nil {
			return d.el.execute(p)
		}

		return "", nil
	}
}

func (d *forDirect) execute(p Params) (string, error) {
	var (
		str string
		err error
		v   reflect.Value
	)
	v, err = d.x.execute(p)
	if err != nil {
		return "", nil
	}
	sb := &strings.Builder{}
	v = uncoverInterface(v)
	np := cop(p)
	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if d.key != nil {
				np[d.key.name.value] = iter.Key().Interface()
			}
			np[d.value.name.value] = iter.Value().Interface()
			if str, err = d.body.execute(np); err != nil {
				return "", err
			} else {
				sb.WriteString(str)
			}
		}

	case reflect.Slice, reflect.Array, reflect.String:
		for i := 0; i < v.Len(); i++ {
			if d.key != nil {
				np[d.key.name.value] = i
			}
			np[d.value.name.value] = v.Index(i).Interface()
			if str, err = d.body.execute(np); err != nil {
				return "", err
			} else {
				sb.WriteString(str)
			}
		}

	default:
		return "", errors.Errorf("can't iter type %s", v.Type())
	}

	return sb.String(), nil
}

func (d *blockDirect) execute(p Params) (string, error) {
	sb := &strings.Builder{}
	var (
		str string
		err error
	)
	if d.body != nil {
		for _, v := range d.body.list {
			if str, err = v.execute(p); err != nil {
				return "", err
			} else {
				sb.WriteString(str)
			}
		}
	}

	if b := p.getBlock(d.name.value.value); b != nil && b != d {
		np := cop(p)
		np.setBlockRemains(sb.String())

		return b.execute(np)
	}

	return sb.String(), nil
}

func (d *includeDirect) execute(p Params) (string, error) {
	if d.params != nil {
		val, err := d.params.execute(p)
		if err != nil {
			return "", err
		}
		if val.Type() != reflect.TypeOf(p) {
			return "", errors.Errorf("can't use type %s as params", val.Type())
		}
		np := val.Interface().(Params)
		if d.only {
			return d.doc.execute(np)
		}
		np = cop(p)
		for k, v := range val.Interface().(Params) {
			np[k] = v
		}

		return d.doc.execute(np)
	}

	return d.doc.execute(p)
}

func (d *extendDirect) execute(p Params) (string, error) {
	panic("unreachable")
}

func boolValue(v reflect.Value) (bool, error) {
	v = uncoverInterface(v)
	var truth bool
	switch v.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		truth = !v.IsZero()

	case reflect.Bool:
		truth = v.Bool()

	case reflect.Pointer:
		truth = !v.IsNil()

	case reflect.Map, reflect.Array, reflect.Slice:
		truth = v.Len() != 0

	default:
		return false, errors.Errorf("can't use %s as condition expression", v.Kind())

	}

	return truth, nil
}

func strValue(v reflect.Value) (string, error) {
	v = uncoverInterface(v)
	kind := v.Kind()
	if isIntLike(kind) {
		return strconv.Itoa(int(v.Int())), nil
	}
	if isUintLike(kind) {
		return strconv.Itoa(int(v.Uint())), nil
	}
	if isFloat(kind) {
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	}
	if kind == reflect.String {
		return v.String(), nil
	}
	if kind == reflect.Bool {
		if vi := v.Bool(); vi {
			return "true", nil
		}
		return "false", nil
	}

	return "", errors.Errorf("can't convert type %s to string", v.Type())
}

func trimString(str string) string {
	if str == "" {
		return str
	}

	return strings.Trim(str, "\"'")
}
