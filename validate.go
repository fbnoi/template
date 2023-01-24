package template

import (
	"reflect"

	"github.com/pkg/errors"
)

var (

	// expr type
	identType      = reflect.TypeOf(&ident{}).Elem()
	indexExprType  = reflect.TypeOf(&indexExpr{}).Elem()
	listExprType   = reflect.TypeOf(&listExpr{}).Elem()
	callExprType   = reflect.TypeOf(&callExpr{}).Elem()
	binaryExprType = reflect.TypeOf(&binaryExpr{}).Elem()

	//direct type
	sectionDirectType = reflect.TypeOf(&sectionDirect{}).Elem()
	blockDirectType   = reflect.TypeOf(&blockDirect{}).Elem()
	ifDirectType      = reflect.TypeOf(&ifDirect{}).Elem()
	extendDirectType  = reflect.TypeOf(&extendDirect{}).Elem()
)

// ----------------------------------------------------------------------------
// ExprNode validation

func (e *ident) validate() error {
	if !goodName(e.name.value) {
		return newUnexpectedToken(e.name)
	}

	return nil
}

func (e *basicLit) validate() error {
	return nil
}

func (e *listExpr) validate() error {
	for _, v := range e.list {
		if err := v.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (e *indexExpr) validate() error {
	switch e.op.value {
	case ".":
		if !isType(e.index, identType, callExprType) {
			return exprValidateError(e)
		}
	case "[":
		if isType(e.index, listExprType) {
			return exprValidateError(e)
		}
	default:
		return newUnexpectedToken(e.op)
	}

	return reportValidateError(e.x.validate, e.index.validate)
}

func (e *callExpr) validate() error {
	return reportValidateError(e.fn.validate, e.args.validate)
}

func (e *binaryExpr) validate() error {
	if isType(e.x, listExprType) || isType(e.y, listExprType) {
		return exprValidateError(e)
	}

	return reportValidateError(e.x.validate, e.y.validate)
}

func (e *singleExpr) validate() error {

	if !isType(e.x, identType, indexExprType) {
		return exprValidateError(e)
	}

	return e.x.validate()
}

func (e *pipelineExpr) validate() error {
	if isType(e.x, listExprType) || (!isType(e.y, identType) && !isType(e.y, callExprType)) {
		return exprValidateError(e)
	}

	return reportValidateError(e.x.validate, e.y.validate)
}

// ----------------------------------------------------------------------------
// DirectNode

func (d *assignDirect) validate() error {
	return reportValidateError(d.lh.validate, d.rh.validate)
}

func (d *sectionDirect) validate() (err error) {
	for _, v := range d.list {
		if err = v.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (d *textDirect) validate() error {
	return d.text.validate()
}

func (d *valueDirect) validate() error {
	return d.tok.validate()
}

func (d *ifDirect) validate() error {
	if !isType(d.cond, identType, indexExprType, callExprType, binaryExprType) {
		return exprValidateError(d.cond)
	}

	for _, v := range d.body.list {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", d.el.typ())
		}
	}

	if err := reportValidateError(d.cond.validate, d.body.validate); err != nil {
		return err
	}

	if d.el != nil {
		if !isType(d.el, ifDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", d.el.typ())
		}

		return d.el.validate()
	}

	return nil
}

func (d *forDirect) validate() error {
	if d.key != nil {
		if err := d.key.validate(); err != nil {
			return exprValidateError(d.key)
		}
	}

	if !isType(d.x, identType, indexExprType, callExprType) {
		return exprValidateError(d.x)
	}

	for _, v := range d.body.list {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.typ())
		}
	}

	if err := reportValidateError(d.value.validate, d.x.validate); err != nil {
		return err
	}

	return d.body.validate()
}

func (d *blockDirect) validate() error {
	for _, v := range d.body.list {
		if isType(v, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.typ())
		}
	}

	return reportValidateError(d.name.validate, d.body.validate)
}

func (d *includeDirect) validate() error {
	if err := d.path.validate(); err != nil {
		return err
	}

	if d.doc.extend != nil {
		return errors.New("con't use extend direct in included template")
	}

	for _, v := range d.doc.body.list {
		if isType(v, blockDirectType, extendDirectType, sectionDirectType) {
			return errors.Errorf("expected %s", v.typ())
		}
	}

	return d.doc.body.validate()
}

func (d *extendDirect) validate() error {
	if err := d.path.validate(); err != nil {
		return err
	}

	for _, v := range d.doc.body.list {
		if isType(v, extendDirectType) {
			return errors.Errorf("expected %s", v.typ())
		}
	}

	return d.doc.body.validate()
}

func reportValidateError(fns ...func() error) (err error) {
	for _, fn := range fns {
		if err = fn(); err != nil {
			return err
		}
	}

	return nil
}

func isType(expr node, typeList ...reflect.Type) bool {
	for _, typ := range typeList {
		if reflect.TypeOf(expr) == typ {
			return true
		}
	}

	return false
}

func exprValidateError(e expr) error {
	return errors.Errorf("parse expr failed: %s", e.literal())
}
