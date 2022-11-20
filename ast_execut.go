package template

import (
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

func (e *Ident) Execute(p *Params) (any, error) {
	return Get(e.Name, p)
}

func (e *BasicLit) Execute(*Params) (any, error) {
	return e.Value, nil
}

func (e *ListExpr) Execute(*Params) (any, error) {
	return nil, nil
}

func (e *OpLit) Execute(*Params) (any, error) {
	return e.Op, nil
}

func (e *IndexExpr) Execute(p *Params) (any, error) {
	x, err := e.X.Execute(p)

	if err != nil {
		return nil, err
	}

	op := e.Op.Op
	switch op {
	case ".":
		switch index := e.Index.(type) {
		case *Ident:
			return Get(index.Name, x)
		case *CallExpr:
			args := []any{}
			for _, v := range index.Args.List {
				if arg, err := v.Execute(p); err == nil {
					args = append(args, arg)
				} else {
					return nil, err
				}
			}

			return callFunc(index.Func.Name, reflect.ValueOf(x), args...)

		default:
			return nil, errors.Errorf("unexpected index token %s", op)
		}
	case "[":
		v, err := e.Index.Execute(p)
		if err != nil {
			return nil, err
		}
		if i, ok := v.(int); ok {
			return Get(strconv.Itoa(i), x)
		}

		if s, ok := v.(string); ok {
			return Get(s, x)
		}

		return nil, errors.Errorf("con't convert %s(type of %s) to type string",
			v,
			reflect.TypeOf(v).Name(),
		)

	default:
		return nil, errors.Errorf("unexpected index token %s", op)
	}
}

func (e *CallExpr) Execute(p *Params) (any, error) {
	args := []any{}
	for _, v := range e.Args.List {
		if arg, err := v.Execute(p); err != nil {
			return nil, err
		} else {
			args = append(args, arg)
		}
	}
	// Fix me: call global func
	return callFunc(e.Func.Name, reflect.ValueOf(e), args...)
}

func (e *BinaryExpr) Execute(p *Params) (any, error) {
	op := e.Op.Op
	x, err := e.X.Execute(p)
	if err != nil {
		return nil, err
	}
	y, err := e.Y.Execute(p)
	if err != nil {
		return nil, err
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
		return x == y, nil
	case "!=":
		return x != y, nil
	}

	return nil, errors.Errorf("unexpected index token %s", op)
}

func (e *SingleExpr) Execute(p *Params) (any, error) {
	x, err := e.X.Execute(p)
	if err != nil {
		return nil, err
	}
	switch e.Op.Op {
	case "not":
		return !reflect.ValueOf(x).IsZero(), nil
	}

	return nil, errors.Errorf("unexpected token %s", e.Op.Op)
}

func greater(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai > bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "compare")
}

func greaterOrEqual(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai >= bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "compare")
}

func add(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai + bi, nil
		}
	case string:
		if bi, ok := b.(string); ok {
			return ai + bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "add")
}

func sub(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai - bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "sub")
}

func multiple(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai + bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "multiple")
}

func divide(a, b any) (any, error) {
	switch ai := a.(type) {
	case int:
		if bi, ok := b.(int); ok {
			return ai + bi, nil
		}
	}

	return nil, binaryTypeError(a, b, "divide")
}

func binaryTypeError(a, b any, op string) error {
	return errors.Errorf("con't %s type %s and type %s",
		op,
		reflect.TypeOf(a).Name(),
		reflect.TypeOf(b).Name(),
	)
}
