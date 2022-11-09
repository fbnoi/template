package template

import (
	"errors"
	"sync"
)

var (
	operator = [...]string{
		"+", "-", "*", "/",
		">", "<", ">=", "<=", "==",
		"|", "or", "and", "in",
	}

	internal_key = [...]string{
		"block", "endblock", "set",
		"if", "elseif", "else", "endif",
		"for", "endfor", "range", "endrange",
		"extend", "include",
	}
)

func Assemble(stream *TokenStream) (doc *Document, err error) {

	// var (
	// 	token *Token
	// 	node  Stmt
	// 	stack []*ASTNode
	// )
	// for !stream.IsEOF() {
	// 	token, err = stream.Next()
	// 	if err != nil {
	// 		return
	// 	}
	// 	switch token.Type() {
	// 	case TYPE_TEXT:
	// 		if  {

	// 		}
	// 	}
	// }

	return
}

type SandBox struct {
	stream *TokenStream
	cursor AppendAble
	stack  []AppendAble
	err    error
	doc    *Document
	pool   sync.Pool
}

func (sb *SandBox) Parse(stream *TokenStream) *SandBox {
	sb.stream = stream
	sb.err = sb.parse()

	return sb
}
func (sb *SandBox) Error() error {
	return sb.err
}
func (sb *SandBox) Doc() *Document {
	return sb.doc
}

func (sb *SandBox) parse() error {
	sb.cursor = sb.doc
	for !sb.stream.IsEOF() {

		token, _ := sb.stream.Next()
		switch token.Type() {
		case TYPE_EOF:
			return nil

		case TYPE_TEXT:
			node := &TextDirect{Text: &BasicLit{Kind: TYPE_STRING}}
			sb.cursor.Append(node)

		case TYPE_VAR_START:
			i := sb.stream.CurrentIndex()
			for token.Type() != TYPE_VAR_END {
				token, _ = sb.stream.Next()
			}
			if i >= sb.stream.CurrentIndex() {
				return &UnexpectedToken{Line: token.Line(), token: token.Value()}
			}
			subStream := sb.stream.SubStream(i, sb.stream.CurrentIndex())
			box := sb.eBox()
			defer sb.putEBox(box)
			if box.Parse(subStream); box.Error() != nil {
				return box.Error()
			}
			node := &ValueDirect{Tok: box.Expr()}
			sb.cursor.Append(node)

		case TYPE_COMMAND_START:
			token, _ := sb.stream.Next()
			switch token.Value() {
			case "endblock":
				if _, ok := sb.cursor.(*BlockDirect); !ok {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				sb.popStack()

			case "endif":
				if _, ok := sb.cursor.(*IfDirect); !ok {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				sb.popStack()

			case "endfor":
				if _, ok := sb.cursor.(*ForDirect); !ok {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				sb.popStack()

			case "block":
				token, _ := sb.stream.Next()
				if token.Type() != TYPE_NAME {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				node := &BlockDirect{Name: &BasicLit{Kind: TYPE_STRING, Value: token.Value()}}
				sb.pushStack(node)

			case "set":
				node := &AssignDirect{}
				token, _ = sb.stream.Next()
				if token.Type() != TYPE_NAME {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				node.Lh = &Ident{Name: token.Value()}

				token, _ = sb.stream.Next()
				if token.Value() != "=" {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}

				token, _ = sb.stream.Next()
				i := sb.stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = sb.stream.Next()
				}
				if i >= sb.stream.CurrentIndex() {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				subStream := sb.stream.SubStream(i, sb.stream.CurrentIndex())
				box := sb.eBox()
				defer sb.putEBox(box)
				if box.Parse(subStream); box.Error() != nil {
					return box.Error()
				}
				node.Rh = box.Expr()
				sb.cursor.Append(node)

			case "elseif":
				ifNode, ok := sb.cursor.(*IfDirect)
				if !ok {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				node := &IfDirect{}
				token, _ = sb.stream.Next()
				i := sb.stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = sb.stream.Next()
				}
				if i >= sb.stream.CurrentIndex() {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				subStream := sb.stream.SubStream(i, sb.stream.CurrentIndex())
				box := sb.eBox()
				defer sb.putEBox(box)
				if box.Parse(subStream); box.Error() != nil {
					return box.Error()
				}
				node.Cond = box.Expr()
				ifNode.Else = node
				sb.shiftStack(node)

			case "else":
				_, ok := sb.cursor.(*IfDirect)
				if !ok {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				sb.cursor.(*IfDirect).Else = &SectionDirect{}

			case "if": // if condition
				node := &IfDirect{}
				token, _ = sb.stream.Next()
				i := sb.stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = sb.stream.Next()
				}
				if i >= sb.stream.CurrentIndex() {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				subStream := sb.stream.SubStream(i, sb.stream.CurrentIndex())
				box := sb.eBox()
				defer sb.putEBox(box)
				if box.Parse(subStream); box.Error() != nil {
					return box.Error()
				}

				node.Cond = box.Expr()
				sb.cursor.Append(node)
				sb.pushStack(node)

			case "for": // for key, value in list, for value in list, for key, _ in list
				node := &ForDirect{}
				token, _ = sb.stream.Next()
				if token.Type() != TYPE_NAME {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				pTok, _ := sb.stream.Peek(1)
				if pTok.Value() != "," {
					node.Value = &Ident{Name: token.Value()}
				} else {
					node.Key = &Ident{Name: token.Value()}
					token, _ = sb.stream.Skip(1)
					node.Value = &Ident{Name: token.Value()}
				}
				token, _ = sb.stream.Next()
				if token.Value() != "in" {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				token, _ = sb.stream.Next()
				i := sb.stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = sb.stream.Next()
				}
				if i >= sb.stream.CurrentIndex() {
					return &UnexpectedToken{Line: token.Line(), token: token.Value()}
				}
				subStream := sb.stream.SubStream(i, sb.stream.CurrentIndex())
				box := sb.eBox()
				defer sb.putEBox(box)
				if box.Parse(subStream); box.Error() != nil {
					return box.Error()
				}
				node.X = box.Expr()
				sb.cursor.Append(node)
				sb.pushStack(node)
			}
		}
	}

	return nil
}

func (sb *SandBox) pushStack(node AppendAble) {
	sb.stack = append(sb.stack, node)
	sb.cursor = node
}

func (sb *SandBox) popStack() {
	sb.cursor = sb.stack[len(sb.stack)-1]
	sb.stack = sb.stack[:len(sb.stack)-1]
}

func (sb *SandBox) shiftStack(node AppendAble) {
	sb.cursor = node
}

func (sb *SandBox) eBox() *ExprSandBox {
	box, _ := sb.pool.Get().(*ExprSandBox)
	box.reset()

	return box
}

func (sb *SandBox) putEBox(ps *ExprSandBox) {
	if ps != nil {
		sb.pool.Put(ps)
	}
}

type ExprSandBox struct {
	sandBox *SandBox
	stream  *TokenStream
	err     error
	expr    Expr
}

func (esb *ExprSandBox) Parse(stream *TokenStream) *ExprSandBox {
	if stream.Len() == 0 {
		esb.err = errors.New("empty stream")
	}

	return esb
}

func (esb *ExprSandBox) Esb() *ExprSandBox {
	return esb.sandBox.eBox()
}

func (esb *ExprSandBox) Error() error {
	return esb.err
}
func (esb *ExprSandBox) Expr() Expr {
	return esb.expr
}
func (esb *ExprSandBox) reset() {
	esb.err = nil
	esb.expr = nil
	esb.stream = nil
}
