package template

import (
	"reflect"

	"github.com/pkg/errors"
)

var (

	// expr type
	identType      = reflect.TypeOf(&Ident{}).Elem()
	indexExprType  = reflect.TypeOf(&IndexExpr{}).Elem()
	ListExprType   = reflect.TypeOf(&ListExpr{}).Elem()
	callExprType   = reflect.TypeOf(&CallExpr{}).Elem()
	binaryExprType = reflect.TypeOf(&BinaryExpr{}).Elem()

	//direct type
	sectionDirectType = reflect.TypeOf(&SectionDirect{}).Elem()
	blockDirectType   = reflect.TypeOf(&BlockDirect{}).Elem()
	ifDirectType      = reflect.TypeOf(&IfDirect{}).Elem()
	extendDirectType  = reflect.TypeOf(&ExtendDirect{}).Elem()
)

// ----------------------------------------------------------------------------
// ExprNode validation
func (e *Ident) Validate() error {
	if !goodName(e.Name.Value()) {
		return newUnexpectedToken(e.Name)
	}

	return nil
}

func (e *BasicLit) Validate() error {
	return nil
}

func (e *ListExpr) Validate() error {
	for _, v := range e.List {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (e *IndexExpr) Validate() error {
	switch e.Op.Value() {
	case ".":
		if !isType(e.Index, identType, callExprType) {
			return exprValidateError(e)
		}
	case "[":
		if isType(e.Index, ListExprType) {
			return exprValidateError(e)
		}
	default:
		return newUnexpectedToken(e.Op)
	}

	return reportValidateError(e.X.Validate, e.Index.Validate)
}

func (e *CallExpr) Validate() error {
	return reportValidateError(e.Func.Validate, e.Args.Validate)
}

func (e *BinaryExpr) Validate() error {
	if isType(e.X, ListExprType) || isType(e.Y, ListExprType) {
		return exprValidateError(e)
	}

	return reportValidateError(e.X.Validate, e.Y.Validate)
}

func (e *SingleExpr) Validate() error {

	if !isType(e.X, identType, indexExprType) {
		return exprValidateError(e)
	}

	return e.X.Validate()
}

// ----------------------------------------------------------------------------
// DirectNode

func (d *AssignDirect) Validate() error {
	return reportValidateError(d.Lh.Validate, d.Rh.Validate)
}

func (d *SectionDirect) Validate() (err error) {
	for _, v := range d.List {
		if err = v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (d *TextDirect) Validate() error {
	return d.Text.Validate()
}

func (d *ValueDirect) Validate() error {
	return d.Tok.Validate()
}

func (d *SetDirect) Validate() error {
	return d.Assign.Validate()
}

func (d *IfDirect) Validate() error {
	if !isType(d.Cond, identType, indexExprType, callExprType, binaryExprType) {
		return exprValidateError(d.Cond)
	}

	for _, v := range d.Body.List {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", d.Else.Type())
		}
	}

	if err := reportValidateError(d.Cond.Validate, d.Body.Validate); err != nil {
		return err
	}

	if d.Else != nil {
		if !isType(d.Else, ifDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", d.Else.Type())
		}

		return d.Else.Validate()
	}

	return nil
}

func (d *ForDirect) Validate() error {
	if d.Key != nil {
		if err := d.Key.Validate(); err != nil {
			return exprValidateError(d.Key)
		}
	}

	if !isType(d.X, identType, indexExprType, callExprType) {
		return exprValidateError(d.X)
	}

	for _, v := range d.Body.List {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.Type())
		}
	}

	if err := reportValidateError(d.Value.Validate, d.X.Validate); err != nil {
		return err
	}

	return d.Body.Validate()
}

func (d *BlockDirect) Validate() error {
	for _, v := range d.Body.List {
		if isType(v, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.Type())
		}
	}

	return reportValidateError(d.Name.Validate, d.Body.Validate)
}

func (d *IncludeDirect) Validate() error {
	if err := d.Path.Validate(); err != nil {
		return err
	}

	if d.Doc.Extend != nil {
		return errors.New("con't use extend direct in included template")
	}

	for _, v := range d.Doc.Body.List {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.Type())
		}
	}

	return d.Doc.Body.Validate()
}

func (d *ExtendDirect) Validate() error {
	if err := d.Path.Validate(); err != nil {
		return err
	}

	for _, v := range d.Doc.Body.List {
		if isType(v, extendDirectType) {
			return errors.Errorf("expected %s", v.Type())
		}
	}

	return d.Doc.Body.Validate()
}

func reportValidateError(fns ...func() error) (err error) {
	for _, fn := range fns {
		if err = fn(); err != nil {
			return err
		}
	}

	return nil
}

func isType(expr Node, typeList ...reflect.Type) bool {
	for _, typ := range typeList {
		if reflect.TypeOf(expr) == typ {
			return true
		}
	}

	return false
}

func exprValidateError(e Expr) error {
	return errors.Errorf("parse expr failed: %s", e.Literal())
}
