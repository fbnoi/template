package template

import (
	"sync"

	"github.com/pkg/errors"
)

var (
	rank = map[string]int{
		".":   0,
		"|":   0,
		"*":   1,
		"/":   1,
		"+":   2,
		"-":   2,
		">":   10,
		"<":   10,
		">=":  10,
		"<=":  10,
		"==":  10,
		"!=":  10,
		"not": 11,
		"or":  12,
		"and": 12,
	}

	internal_key = [...]string{
		"block", "endblock", "set",
		"if", "elseif", "else", "endif",
		"for", "endfor", "range", "endrange",
		"extend", "include",
	}
)

func Assemble(stream *TokenStream) (doc *Document, err error) {
	doc = &Document{}
	sb := &SandBox{doc: doc, pool: sync.Pool{
		New: func() interface{} {
			return &ExprSandBox{}
		},
	}}
	err = sb.Parse(stream).err

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
			node := &TextDirect{Text: &BasicLit{Kind: TYPE_STRING, Value: token.Value()}}
			sb.cursor.Append(node)

		case TYPE_VAR_START:
			i := sb.stream.CurrentIndex() + 1
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

			default:
				return &UnexpectedToken{Line: token.Line(), token: token.Value()}
			}
		}
	}

	return nil
}

func (sb *SandBox) pushStack(node AppendAble) {
	sb.stack = append(sb.stack, sb.cursor)
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

	exprStack []Expr
	opStack   []*Token
}

func (esb *ExprSandBox) Parse(stream *TokenStream) *ExprSandBox {
	if stream.Len() == 0 {
		esb.err = errors.New("empty stream")
		return esb
	}

	for !stream.IsEOF() {
		if esb.err != nil {
			return esb
		}
		token, _ := stream.Next()
		switch token.Type() {
		case TYPE_NUMBER, TYPE_STRING:
			b := &BasicLit{Kind: token.Type(), Value: token.value}
			esb.exprStack = append(esb.exprStack, b)

		case TYPE_NAME:
			i := &Ident{Name: token.Value()}
			if !stream.IsEOF() {
				if nextToken, err := stream.Peek(1); err == nil && nextToken.Value() == "(" {
					c := &CallExpr{Func: i}
					esb.exprStack = append(esb.exprStack, c)
					continue
				}
			}
			esb.exprStack = append(esb.exprStack, i)

		case TYPE_OPERATOR:
			if !allowOp(token) {
				esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}
			}
			if len(esb.opStack) == 0 {
				esb.opStack = append(esb.opStack, token)
			} else {
				topOp := esb.opStack[len(esb.opStack)-1]
				for compare(topOp.Value(), token.Value()) {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.opStack = append(esb.opStack, token)
			}

		case TYPE_PUNCTUATION:
			switch token.Value() {
			case "(", "[":
				esb.opStack = append(esb.opStack, token)

			case ")":
				topOp := esb.opStack[len(esb.opStack)-1]
				for topOp.Value() != "(" {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.opStack = esb.opStack[:len(esb.opStack)-1]
				esb.mergeExprStack(token)

			case "]":
				topOp := esb.opStack[len(esb.opStack)-1]
				for topOp.Value() != "[" {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.opStack = esb.opStack[:len(esb.opStack)-1]
				esb.mergeExprStack(token)

			case ",":
				topOp := esb.opStack[len(esb.opStack)-1]
				for topOp.Value() != "(" {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.mergeExprStack(token)
			default:
				esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}

			}
		case TYPE_EOF:
		default:
			esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}
		}
	}

	for i := len(esb.opStack) - 1; i >= 0; i-- {
		topOp := esb.opStack[i]
		esb.opStack = esb.opStack[:i]
		if !esb.mergeExprStack(topOp) {
			return esb
		}
	}

	if len(esb.exprStack) != 1 {
		esb.err = errors.Errorf("parse expr failed: %s", stream.String())

		return esb
	}

	esb.expr = esb.exprStack[0]

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
func (esb *ExprSandBox) mergeExprStack(token *Token) bool {
	if len(esb.exprStack) < 2 {
		esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}

		return false
	}
	op := &OpLit{Op: token.Value()}

	switch token.Value() {
	case "+", "-", "*", "/", ">", "==", "<", ">=", "<=", "and", "or":
		expr1 := esb.exprStack[len(esb.exprStack)-2]
		expr2 := esb.exprStack[len(esb.exprStack)-1]
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		b := &BinaryExpr{X: expr1, Op: op, Y: expr2}
		esb.exprStack = append(esb.exprStack, b)
	case "not", "++", "--":
		expr1 := esb.exprStack[len(esb.exprStack)-1]
		expr1 = &SingleExpr{X: expr1, Op: op}

	case "[", ".":
		expr1 := esb.exprStack[len(esb.exprStack)-2]
		expr2 := esb.exprStack[len(esb.exprStack)-1]
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		if _, ok := expr1.(*Ident); !ok {
			esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}

			return false
		}
		i := &IndexExpr{X: expr1, Op: op, Index: expr2}
		esb.exprStack = append(esb.exprStack, i)
	case ",":
		expr1 := esb.exprStack[len(esb.exprStack)-1]
		expr2 := esb.exprStack[len(esb.exprStack)-2]
		if list, ok := expr2.(*ListExpr); ok {
			list.List = append(list.List, expr1)
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-1]
		} else {
			list := &ListExpr{}
			list.List = append(list.List, expr1)
			esb.exprStack[len(esb.exprStack)-1] = list
		}
	case ")":
		expr1 := esb.exprStack[len(esb.exprStack)-1]
		expr2 := esb.exprStack[len(esb.exprStack)-2]
		if list, ok := expr2.(*ListExpr); ok {
			list.List = append(list.List, expr1)
			expr3 := esb.exprStack[len(esb.exprStack)-3]
			fn, _ := expr3.(*CallExpr)
			fn.Args = list
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		} else if fn, ok := expr2.(*CallExpr); ok {
			list := &ListExpr{}
			list.List = append(list.List, expr1)
			fn.Args = list
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-1]
		}
	default:
		esb.err = &UnexpectedToken{Line: token.Line(), token: token.Value()}

		return false
	}

	return true
}

func compare(op1, op2 string) bool {
	return rank[op1] < rank[op2]
}

func allowOp(token *Token) bool {
	_, ok := rank[token.Value()]

	return ok
}
