package template

import (
	"fmt"
	"reflect"
	"strings"
)

type node interface {
	validate() error
}

// All expression nodes implement the Expr interface.
type expr interface {
	node
	exprNode()
	literal() string
	execute(p Params) (reflect.Value, error)
}

// All statement nodes implement the Direct interface.
type direct interface {
	node
	directNode()
	execute(p Params) (string, error)
	typ() string
}

type appendAble interface {
	append(direct)
}

// ----------------------------------------------------------------------------
// ExprNode

type (
	ident struct {
		name *token // identifier name; not nil
	}

	basicLit struct {
		kind  int    // type_number or type_string
		value *token // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', etc.; not nil
	}

	listExpr struct {
		list []expr
	}

	// An indexExpr node represents an expression followed by an index.
	indexExpr struct {
		x     expr   // expression; not nil
		index expr   // index expression; not nil
		op    *token // not nil
	}

	// A callExpr node represents an expression followed by an argument list.
	callExpr struct {
		fn   *ident    // function expression; not nil
		args *listExpr // function arguments; or nil
	}

	// A binaryExpr node represents a binary expression.
	binaryExpr struct {
		x  expr   // left operand; not nil
		op *token // operator; not nil
		y  expr   // right operand; not nil
	}

	// A Single node represents a single expression.
	singleExpr struct {
		x  expr   // expr; not nil
		op *token // operator; not nil
	}
)

// exprNode() ensures that only expression/type nodes can be
// assigned to an Expr.
func (*ident) exprNode()      {}
func (*basicLit) exprNode()   {}
func (*listExpr) exprNode()   {}
func (*indexExpr) exprNode()  {}
func (*callExpr) exprNode()   {}
func (*binaryExpr) exprNode() {}
func (*singleExpr) exprNode() {}

// ----------------------------------------------------------------------------
// Statements

type (

	// An assignDirect node represents an assignment or
	// a short variable declaration.
	//
	assignDirect struct {
		lh *ident
		rh expr
	}

	// A sectionDirect node represents a braced statement list.
	sectionDirect struct {
		list []direct
	}

	// textDirect
	textDirect struct {
		text *basicLit // text content basicLit
	}

	valueDirect struct {
		tok expr // value expr
	}

	ifDirect struct {
		cond expr           // condition; not nil
		el   direct         // else branch; or nil
		body *sectionDirect // not nil
	}

	forDirect struct {
		key, value *ident // Key may be nil, Value, ident expr
		x          expr   // value to range over
		body       *sectionDirect
	}

	blockDirect struct {
		name *basicLit      // name of block; not nil
		body *sectionDirect // body of block; not nil
	}

	includeDirect struct {
		path   *basicLit // string of template path
		params expr      // parameters injected into include doc
		doc    *Document // not nil
		only   bool
	}

	extendDirect struct {
		path *basicLit // string of template path
		doc  *Document
	}
)

// directNode() ensures that only statement nodes can be
// assigned to a Direct.
//

func (*textDirect) directNode()    {}
func (*valueDirect) directNode()   {}
func (*assignDirect) directNode()  {}
func (*sectionDirect) directNode() {}
func (*ifDirect) directNode()      {}
func (*forDirect) directNode()     {}
func (*blockDirect) directNode()   {}
func (*includeDirect) directNode() {}
func (*extendDirect) directNode()  {}
func (*Document) directNode()      {}

func (*textDirect) typ() string {
	return "textDirect"
}
func (*valueDirect) typ() string {
	return "valueDirect"
}
func (*assignDirect) typ() string {
	return "assignDirect"
}
func (*sectionDirect) typ() string {
	return "sectionDirect"
}
func (*ifDirect) typ() string {
	return "ifDirect"
}
func (*forDirect) typ() string {
	return "forDirect"
}
func (*blockDirect) typ() string {
	return "blockDirect"
}
func (*includeDirect) typ() string {
	return "includeDirect"
}
func (*extendDirect) typ() string {
	return "extendDirect"
}
func (*Document) typ() string {
	return "Document"
}

// append() ensures that only statement nodes can be
// assigned to a Direct.
func (s *ifDirect) append(x direct) {
	if s.el != nil {
		if _, ok := s.el.(*sectionDirect); ok {
			s.el.(*sectionDirect).list = append(s.el.(*sectionDirect).list, x)
		}
	}
	if s.body == nil {
		s.body = &sectionDirect{}
	}
	s.body.list = append(s.body.list, x)
}

func (s *forDirect) append(x direct) {
	if s.body == nil {
		s.body = &sectionDirect{}
	}
	s.body.list = append(s.body.list, x)
}

func (s *blockDirect) append(x direct) {
	if s.body == nil {
		s.body = &sectionDirect{}
	}
	s.body.list = append(s.body.list, x)
}

func (e *ident) literal() string {
	return e.name.value
}
func (e *basicLit) literal() string {
	return e.value.value
}
func (e *listExpr) literal() string {
	var ts []string
	for _, v := range e.list {
		ts = append(ts, v.literal())
	}

	return strings.Join(ts, ",")
}
func (e *indexExpr) literal() string {

	if e.op.value == "." {
		return fmt.Sprintf("%s.%s", e.x.literal(), e.index.literal())
	}

	if e.op.value == "[" {
		return fmt.Sprintf("%s[%s]", e.x.literal(), e.index.literal())
	}

	return "<indexExpr ParseError>"
}
func (e *callExpr) literal() string {
	return fmt.Sprintf("%s(%s)", e.fn.literal(), e.args.literal())
}
func (e *binaryExpr) literal() string {
	return fmt.Sprintf("%s %s %s", e.x.literal(), e.op.value, e.y.literal())
}
func (e *singleExpr) literal() string {
	return fmt.Sprintf("%s %s", e.op.value, e.x.literal())
}
