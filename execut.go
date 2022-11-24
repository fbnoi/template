package template

import (
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

func (e *Ident) Execute(p *Params) (reflect.Value, error) {
	return Get(p, e.Name.Value())
}

func (e *BasicLit) Execute(*Params) (reflect.Value, error) {
	vs := e.Value.Value()
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

func (e *ListExpr) Execute(*Params) (reflect.Value, error) {
	return zeroValue, nil
}

func (e *IndexExpr) Execute(p *Params) (reflect.Value, error) {
	x, err := e.X.Execute(p)
	if err != nil {
		return zeroValue, err
	}
	vx := x.Interface()
	op := e.Op.Value()
	switch op {
	case ".":
		switch index := e.Index.(type) {
		case *Ident:
			return Get(vx, index.Name.Value())
		case *CallExpr:
			if fn, err := method(x, index.Func.Name.Value()); err != nil {
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

func (e *CallExpr) Execute(p *Params) (reflect.Value, error) {
	if fn, err := method(reflect.ValueOf(p), e.Func.Name.Value()); err != nil {
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

func (e *BinaryExpr) Execute(p *Params) (reflect.Value, error) {
	op := e.Op.Value()
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

func (e *SingleExpr) Execute(p *Params) (reflect.Value, error) {
	x, err := e.X.Execute(p)
	if err != nil {
		return zeroValue, err
	}

	switch e.Op.Value() {
	case "not":
		r := x.IsZero()

		return reflect.ValueOf(!r), nil
	}

	return zeroValue, newUnexpectedToken(e.Op)
}
