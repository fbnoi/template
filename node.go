package template

import (
	"fmt"
	"reflect"
	"strings"
)

const NoPos Pos = 0

type Pos int

func (p Pos) Position() Pos {
	return p
}

type Node interface {
	Validate() error
}

// All expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
	Literal() string
	Execute(p *Params) (reflect.Value, error)
}

// All statement nodes implement the Direct interface.
type Direct interface {
	Node
	directNode()
}

type AppendAble interface {
	Append(Direct)
}

// ----------------------------------------------------------------------------
// ExprNode

type (
	Ident struct {
		Name *Token // identifier name; not nil
	}

	BasicLit struct {
		Kind  int    // TYPE_NUMBER or TYPE_STRING
		Value *Token // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', etc.; not nil
	}

	ListExpr struct {
		List []Expr
	}

	// An IndexExpr node represents an expression followed by an index.
	IndexExpr struct {
		X     Expr   // expression; not nil
		Index Expr   // index expression; not nil
		Op    *Token // not nil
	}

	// A CallExpr node represents an expression followed by an argument list.
	CallExpr struct {
		Func *Ident    // function expression; not nil
		Args *ListExpr // function arguments; or nil
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X  Expr   // left operand; not nil
		Op *Token // operator; not nil
		Y  Expr   // right operand; not nil
	}

	// A Single node represents a single expression.
	SingleExpr struct {
		X  Expr   // expr; not nil
		Op *Token // operator; not nil
	}
)

// exprNode() ensures that only expression/type nodes can be
// assigned to an Expr.
//
func (*Ident) exprNode()      {}
func (*BasicLit) exprNode()   {}
func (*ListExpr) exprNode()   {}
func (*IndexExpr) exprNode()  {}
func (*CallExpr) exprNode()   {}
func (*BinaryExpr) exprNode() {}
func (*SingleExpr) exprNode() {}

// ----------------------------------------------------------------------------
// Statements

type (

	// An AssignDirect node represents an assignment or
	// a short variable declaration.
	//
	AssignDirect struct {
		Lh *Ident
		Rh Expr
	}

	// A SectionDirect node represents a braced statement list.
	SectionDirect struct {
		List []Direct
	}

	// TextDirect
	TextDirect struct {
		Text *BasicLit // text content BasicLit
	}

	ValueDirect struct {
		Tok Expr // value expr
	}

	SetDirect struct {
		Assign *AssignDirect
	}

	// An IfDirect node represents an if statement.
	IfDirect struct {
		Cond Expr           // condition; not nil
		Else Direct         // else branch; or nil
		Body *SectionDirect // not nil
	}

	// A ForDirect represents a for statement.
	ForDirect struct {
		Key, Value *Ident // Key, Value may be nil, Ident expr
		X          Expr   // value to range over
		Body       *SectionDirect
	}

	//
	BlockDirect struct {
		Name *BasicLit      // name of block; not nil
		Body *SectionDirect // body of block; not nil
	}

	IncludeDirect struct {
		Path   *BasicLit // string of template path
		Params Params    // parameters injected into block
		Doc    *Document // not nil
	}

	ExtendDirect struct {
		Path *BasicLit // string of template path
		Doc  *Document
	}
)

// DirectNode() ensures that only statement nodes can be
// assigned to a Direct.
//

func (*TextDirect) directNode()    {}
func (*ValueDirect) directNode()   {}
func (*AssignDirect) directNode()  {}
func (*SectionDirect) directNode() {}
func (*IfDirect) directNode()      {}
func (*ForDirect) directNode()     {}
func (*BlockDirect) directNode()   {}
func (*IncludeDirect) directNode() {}
func (*ExtendDirect) directNode()  {}
func (*SetDirect) directNode()     {}

// Append() ensures that only statement nodes can be
// assigned to a Direct.
//
func (s *IfDirect) Append(x Direct) {
	if s.Else != nil {
		if _, ok := s.Else.(*SectionDirect); ok {
			s.Else.(*SectionDirect).List = append(s.Else.(*SectionDirect).List, x)
		}
	}
	if s.Body == nil {
		s.Body = &SectionDirect{}
	}
	s.Body.List = append(s.Body.List, x)
}

func (s *ForDirect) Append(x Direct) {
	if s.Body == nil {
		s.Body = &SectionDirect{}
	}
	s.Body.List = append(s.Body.List, x)
}

func (s *BlockDirect) Append(x Direct) {
	if s.Body == nil {
		s.Body = &SectionDirect{}
	}
	s.Body.List = append(s.Body.List, x)
}

func (e *Ident) Literal() string {
	return e.Name.value
}
func (e *BasicLit) Literal() string {
	return e.Value.value
}
func (e *ListExpr) Literal() string {
	var ts []string
	for _, v := range e.List {
		ts = append(ts, v.Literal())
	}

	return strings.Join(ts, ",")
}
func (e *IndexExpr) Literal() string {

	if e.Op.value == "." {
		return fmt.Sprintf("%s.%s", e.X.Literal(), e.Index.Literal())
	}

	if e.Op.value == "[" {
		return fmt.Sprintf("%s[%s]", e.X.Literal(), e.Index.Literal())
	}

	return "<IndexExpr ParseError>"
}
func (e *CallExpr) Literal() string {
	return fmt.Sprintf("%s(%s)", e.Func.Literal(), e.Args.Literal())
}
func (e *BinaryExpr) Literal() string {
	return fmt.Sprintf("%s%s%s", e.X.Literal(), e.Op.value, e.Y.Literal())
}
func (e *SingleExpr) Literal() string {
	return fmt.Sprintf("%s %s", e.Op.value, e.X.Literal())
}
