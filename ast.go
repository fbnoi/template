package template

const NoPos Pos = 0

type Pos int

func (p Pos) Position() Pos {
	return p
}

type ASTNode interface {
}

// All expression nodes implement the Expr interface.
type Expr interface {
	ASTNode
	exprNode()
}

// All statement nodes implement the Direct interface.
type Direct interface {
	ASTNode
	directNode()
}

// All text nodes implement the Text interface.
type Text interface {
	ASTNode
	textNode()
}

type AppendAble interface {
	Append(Direct)
}

// ----------------------------------------------------------------------------
// Expressions

type (
	Ident struct {
		Name string // identifier name
	}

	BasicLit struct {
		Kind  int    // TYPE_NUMBER, TYPE_STRING
		Value string // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', etc.
	}

	ListExpr struct {
		List []Expr
	}

	OpLit struct {
		Op string // literal string; e.g. + - * /
	}

	// An IndexExpr node represents an expression followed by an index.
	IndexExpr struct {
		X     Expr // expression
		Index Expr // index expression
		Op    *OpLit
	}

	// A CallExpr node represents an expression followed by an argument list.
	CallExpr struct {
		Func *Ident    // function expression
		Args *ListExpr // function arguments; or nil
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X  Expr   // left operand
		Op *OpLit // operator
		Y  Expr   // right operand
	}

	// A Single node represents a single expression.
	SingleExpr struct {
		X  Expr   // expr
		Op *OpLit // operator
	}
)

// exprNode() ensures that only expression/type nodes can be
// assigned to an Expr.
//
func (*Ident) exprNode()      {}
func (*BasicLit) exprNode()   {}
func (*ListExpr) exprNode()   {}
func (*OpLit) exprNode()      {}
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
		Lh Expr
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
		Cond Expr   // condition
		Else Direct // else branch; or nil
		Body *SectionDirect
	}

	// A ForDirect represents a for statement.
	ForDirect struct {
		Key, Value Expr // Key, Value may be nil, Ident expr
		X          Expr // value to range over
		Body       *SectionDirect
	}

	//
	BlockDirect struct {
		Name *BasicLit      // name of block
		Body *SectionDirect // body of block
	}

	IncludeDirect struct {
		Ident  *BasicLit      // string of block name
		Params map[string]any // parameters injected into block
		Doc    *Document
	}

	ExtendDirect struct {
		Ident *BasicLit // string of block name
		Doc   *Document
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
