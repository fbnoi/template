package template

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func (e *Ident) Execute(p Params) (reflect.Value, error) {
	return Get(p, e.Name.value)
}

func (e *BasicLit) Execute(Params) (reflect.Value, error) {
	vs := e.Value.value
	if e.Kind == TYPE_STRING {
		return reflect.ValueOf(vs), nil
	}
	if e.Kind == TYPE_NUMBER {
		if i, err := strconv.Atoi(vs); err == nil {
			return reflect.ValueOf(i), nil
		}
		if f, err := strconv.ParseFloat(vs, 64); err == nil {
			return reflect.ValueOf(f), nil
		}
	}

	return zeroValue, newUnexpectedToken(e.Value)
}

func (e *ListExpr) Execute(Params) (reflect.Value, error) {
	return zeroValue, nil
}

func (e *IndexExpr) Execute(p Params) (reflect.Value, error) {
	x, err := e.X.Execute(p)
	if err != nil {
		return zeroValue, err
	}
	vx := x.Interface()
	op := e.Op.value
	switch op {
	case ".":
		switch index := e.Index.(type) {
		case *Ident:
			return Get(vx, index.Name.value)
		case *CallExpr:
			if fn, err := method(x, index.Func.Name.value); err != nil {
				return zeroValue, err
			} else {
				argv := []reflect.Value{}
				for _, v := range index.Args.List {
					if arg, err := v.Execute(p); err == nil {
						argv = append(argv, arg)
					} else {
						return zeroValue, err
					}
				}

				return call(fn, argv...)
			}

		default:
			return zeroValue, newUnexpectedToken(e.Op)

		}

	case "[":
		v, err := e.Index.Execute(p)
		if err != nil {
			return zeroValue, err
		}
		if v.CanInt() {
			return Get(vx, x)
		}
		if v.Kind() == reflect.String {
			return Get(vx, v.Interface().(string))
		}
		return zeroValue, errors.Errorf("con't convert %s(type of %s) to type string",
			v,
			reflect.TypeOf(v).Name(),
		)

	default:
		return zeroValue, errors.Errorf("unexpected index token %s", op)
	}
}

func (e *CallExpr) Execute(p Params) (reflect.Value, error) {
	if fn, err := method(reflect.ValueOf(p), e.Func.Name.value); err != nil {
		return zeroValue, err
	} else {
		argv := []reflect.Value{}
		for _, v := range e.Args.List {
			if arg, err := v.Execute(p); err == nil {
				argv = append(argv, arg)
			} else {
				return zeroValue, err
			}
		}

		return call(fn, argv...)
	}
}

func (e *BinaryExpr) Execute(p Params) (reflect.Value, error) {
	op := e.Op.value
	x, err := e.X.Execute(p)
	if err != nil {
		return zeroValue, err
	}
	y, err := e.Y.Execute(p)
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
	}

	return zeroValue, newUnexpectedToken(e.Op)
}

func (e *SingleExpr) Execute(p Params) (reflect.Value, error) {
	x, err := e.X.Execute(p)
	if err != nil {
		return zeroValue, err
	}
	switch e.Op.value {
	case "not":
		r := x.IsZero()

		return reflect.ValueOf(!r), nil
	}

	return zeroValue, newUnexpectedToken(e.Op)
}

func (d *TextDirect) Execute(p Params) (string, error) {
	return d.Text.Value.value, nil
}

func (d *ValueDirect) Execute(p Params) (string, error) {
	if v, err := d.Tok.Execute(p); err != nil {
		return "", err
	} else {
		return strValue(v)
	}
}

func (d *AssignDirect) Execute(p Params) (string, error) {
	yx, err := d.Rh.Execute(p)
	if err != nil {
		return "", err
	}
	p[d.Lh.Name.value] = yx

	return "", nil
}

func (d *SectionDirect) Execute(p Params) (string, error) {
	sb := &strings.Builder{}
	for _, x := range d.List {
		if str, err := x.Execute(p); err != nil {
			return "", err
		} else {
			sb.WriteString(str)
		}
	}

	return sb.String(), nil
}

func (d *IfDirect) Execute(p Params) (string, error) {
	if conv, err := d.Cond.Execute(p); err != nil {
		return "", err
	} else {
		conv = uncoverInterface(conv)
		if conv.IsNil() || conv.IsZero() {
			if d.Else != nil {
				return d.Else.Execute(p)
			}

			return "", nil
		} else {
			return d.Body.Execute(p)
		}
	}
}

func (d *ForDirect) Execute(p Params) (string, error) {
	var (
		str string
		err error
		v   reflect.Value
	)
	v, err = d.X.Execute(p)
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
			if d.Key != nil {
				np[d.Key.Name.value] = iter.Key()
			}
			np[d.Key.Name.value] = iter.Value()
			if str, err = d.Body.Execute(np); err != nil {
				return "", err
			} else {
				sb.WriteString(str)
			}
		}

	case reflect.Slice, reflect.Array, reflect.String:
		for i := 0; i < v.Len(); i++ {
			if d.Key != nil {
				np[d.Key.Name.value] = i
			}
			np[d.Key.Name.value] = v.Index(i)
			if str, err = d.Body.Execute(np); err != nil {
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

func (d *BlockDirect) Execute(p Params) (string, error) {
	sb := &strings.Builder{}
	var (
		str string
		err error
	)
	for _, v := range d.Body.List {
		if str, err = v.Execute(p); err != nil {
			return "", err
		} else {
			sb.WriteString(str)
		}
	}

	if b := p.getBlock(d.Name.Value.value); b != nil && b != d {
		np := cop(p)
		np.setBlockRemains(sb.String())

		return b.Execute(np)
	}

	return sb.String(), nil
}

func (d *IncludeDirect) Execute(p Params) (string, error) {
	if d.Params != nil {
		val, err := d.Params.Execute(p)
		if err != nil {
			return "", err
		}
		if val.Type() != reflect.TypeOf(p) {
			return "", errors.Errorf("con't use type %s as params", val.Type())
		}
		if d.Only {
			return d.Doc.Body.Execute(val.Interface().(Params))
		}
		np := cop(p)
		for k, v := range val.Interface().(Params) {
			np[k] = v
		}

		return d.Doc.Body.Execute(np)
	}

	return d.Doc.Body.Execute(p)
}

func (d *ExtendDirect) Execute(p Params) (string, error) {
	panic("unreachable")
}

func (d *SetDirect) Execute(p Params) (string, error) {
	return d.Assign.Execute(p)
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
