package template

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	rank = map[string]int{
		".":   1,
		"*":   3,
		"/":   3,
		"+":   4,
		"-":   4,
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

	internalKeyWords = "_block_endblock_set_if_elseif_else_endif_for_endfor_extend_include_in_and_or_not_with_"

	sandboxPool = sync.Pool{
		New: func() any {
			return &sandbox{}
		},
	}

	exprSandboxPool = sync.Pool{
		New: func() any {
			return &exprSandbox{}
		},
	}
)

func BuildTemplate(content string) (*Document, error) {
	var (
		source *Source
		err    error
	)
	source = NewSource(content)
	if err != nil {
		return nil, err
	}

	return buildSource(source)
}

func BuildFileTemplate(path string) (doc *Document, err error) {
	if doc = _cache.Doc(path); doc != nil {
		return
	}
	var source *Source
	source, err = NewSourceFile(path)
	if err != nil {
		return nil, err
	}

	doc, err = buildSource(source)
	if err != nil {
		return
	}
	_cache.AddDoc(path, doc)

	return
}

func buildSource(source *Source) (*Document, error) {
	var (
		stream *TokenStream
		err    error
	)
	if err != nil {
		return nil, err
	}
	stream, err = Tokenize(source)
	if err != nil {
		return nil, err
	}
	doc := NewDocument()
	err = build(doc, stream)

	return doc, err
}

func build(doc *Document, stream *TokenStream) error {
	sb := getSandbox()
	defer putSandbox(sb)
	err := sb.build(doc, stream)

	return err
}

func compare(op1, op2 string) bool {
	return rank[op1] <= rank[op2]
}

func allowOp(op *Token) bool {
	_, ok := rank[op.value]

	return ok
}

func getSandbox() *sandbox {
	return sandboxPool.Get().(*sandbox)
}

func putSandbox(sb *sandbox) {
	sb.reset()
	sandboxPool.Put(sb)
}

func getExprSandbox() *exprSandbox {
	return exprSandboxPool.Get().(*exprSandbox)
}

func putExprSandbox(sb *exprSandbox) {
	sb.reset()
	exprSandboxPool.Put(sb)
}

type sandbox struct {
	cursor AppendAble
	stack  []AppendAble
}

func (sb *sandbox) build(doc *Document, stream *TokenStream) error {
	sb.cursor = doc
	var (
		token     *Token
		err       error
		node      Direct
		subStream *TokenStream
		box       *exprSandbox
		ok        bool
		baseDoc   *Document
		boxes     []*exprSandbox
	)

	defer func(boxes []*exprSandbox) {
		for _, box := range boxes {
			putExprSandbox(box)
		}
	}(boxes)

	for stream.HasNext() {
		if token, err = stream.Next(); err != nil {
			return err
		}

		switch token.Type() {
		case TYPE_EOF:
			return nil

		case TYPE_TEXT:
			node = &TextDirect{Text: &BasicLit{Kind: TYPE_STRING, Value: token}}
			sb.cursor.Append(node)

		case TYPE_VAR_START:
			if subStream, err = subStreamIf(stream, func(t *Token) bool {
				return t.typ != TYPE_VAR_END
			}); err != nil {
				return err
			} else {
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node = &ValueDirect{Tok: box.expr}
				sb.cursor.Append(node)
			}

		case TYPE_COMMAND_START:
			token, err = stream.Next()
			if err != nil {
				return err
			}
			switch token.value {
			case "endblock":
				if _, ok = sb.cursor.(*BlockDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "endif":
				if _, ok = sb.cursor.(*IfDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "endfor":
				if _, ok = sb.cursor.(*ForDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "extend":
				node = &ExtendDirect{}
				if token, err = nextTokenTypeShouldBe(stream, TYPE_STRING); err != nil {
					return err
				}
				node.(*ExtendDirect).Path = &BasicLit{Kind: token.typ, Value: token}
				if baseDoc, err := BuildFileTemplate(token.value); err != nil {
					return err
				} else {
					baseDoc.extended = true
					node.(*ExtendDirect).Doc = baseDoc
				}
				doc.Extend = node.(*ExtendDirect)

			case "include":
				node = &IncludeDirect{}
				if token, err = nextTokenTypeShouldBe(stream, TYPE_STRING); err != nil {
					return err
				}
				node.(*IncludeDirect).Path = &BasicLit{Kind: token.typ, Value: token}
				if baseDoc, err = BuildFileTemplate(token.value); err != nil {
					return err
				} else {
					node.(*IncludeDirect).Doc = baseDoc
				}
				if token, err = stream.Next(); err != nil {
					return err
				}
				if token.value == "with" {
					if subStream, err = subStreamIf(stream, func(t *Token) bool {
						return t.typ != TYPE_COMMAND_END && t.value != "only"
					}); err != nil {
						return err
					}
					box = getExprSandbox()
					boxes = append(boxes, box)
					if err = box.build(subStream); err != nil {
						return err
					}
					node.(*IncludeDirect).Params = box.expr

					if token, err := stream.Current(); err != nil {
						return err
					} else if token.value == "only" {
						node.(*IncludeDirect).Only = true
					}
				}
				sb.cursor.Append(node)

			case "block":
				if token, err = nextTokenTypeShouldBe(stream, TYPE_NAME); err != nil {
					return err
				}
				node = &BlockDirect{Name: &BasicLit{Kind: TYPE_STRING, Value: token}}
				if _, ok := doc.blocks[token.value]; ok {
					return errors.Errorf("block %s has already exist", token.value)
				}
				doc.blocks[token.value] = node.(*BlockDirect)
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node.(*BlockDirect))

			case "set":
				node = &AssignDirect{}
				if token, err = nextTokenTypeShouldBe(stream, TYPE_NAME); err != nil {
					return err
				}
				node.(*AssignDirect).Lh = &Ident{Name: token}
				if _, err = nextTokenValueShouldBe(stream, "="); err != nil {
					return err
				}
				if subStream, err = subStreamIf(stream, func(t *Token) bool {
					return t.typ != TYPE_COMMAND_END
				}); err != nil {
					return err
				}
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node.(*AssignDirect).Rh = box.expr
				sb.cursor.Append(node)

			case "elseif":
				ifNode, ok := sb.cursor.(*IfDirect)
				if !ok {
					return newUnexpectedToken(token)
				}
				if subStream, err = subStreamIf(stream, func(t *Token) bool {
					return t.typ != TYPE_COMMAND_END
				}); err != nil {
					return err
				}
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node = &IfDirect{}
				node.(*IfDirect).Cond = box.expr
				ifNode.Else = node
				sb.shiftStack(node.(*IfDirect))

			case "else":
				_, ok := sb.cursor.(*IfDirect)
				if !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor.(*IfDirect).Else = &SectionDirect{}

			case "if":
				if subStream, err = subStreamIf(stream, func(t *Token) bool {
					return t.typ != TYPE_COMMAND_END
				}); err != nil {
					return err
				}
				node = &IfDirect{}
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node.(*IfDirect).Cond = box.expr
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node.(*IfDirect))

			case "for":
				if subStream, err = subStreamIf(stream, func(t *Token) bool {
					return t.value != "in"
				}); err != nil {
					return err
				}
				node = &ForDirect{}
				switch subStream.Size() {
				case 2:
					if token, err = nextTokenTypeShouldBe(subStream, TYPE_NAME); err != nil {
						return err
					}
					node.(*ForDirect).Value = &Ident{Name: token}
				case 4:
					if token, err = nextTokenTypeShouldBe(subStream, TYPE_NAME); err != nil {
						return err
					}
					node.(*ForDirect).Key = &Ident{Name: token}
					subStream.Skip(1)
					if token, err = nextTokenTypeShouldBe(subStream, TYPE_NAME); err != nil {
						return err
					}
					node.(*ForDirect).Value = &Ident{Name: token}
				default:
					return errors.Errorf("Unexpected arg list %s in for loop", subStream.String())
				}

				if subStream, err = subStreamIf(stream, func(t *Token) bool {
					return t.typ != TYPE_COMMAND_END
				}); err != nil {
					return err
				}
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node.(*ForDirect).X = box.expr
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node.(*ForDirect))

			default:
				return newUnexpectedToken(token)

			}

		case TYPE_COMMAND_END, TYPE_VAR_END:
			continue

		default:
			return newUnexpectedToken(token)

		}
	}

	return nil
}

func (sb *sandbox) pushStack(node AppendAble) AppendAble {
	sb.stack = append(sb.stack, sb.cursor)

	return node
}

func (sb *sandbox) popStack() AppendAble {
	tmp := sb.stack[len(sb.stack)-1]
	sb.stack = sb.stack[:len(sb.stack)-1]

	return tmp
}

func (sb *sandbox) shiftStack(node AppendAble) {
	sb.cursor = node
}

func (sb *sandbox) reset() {
	sb.cursor = nil
	sb.stack = sb.stack[0:0]
}

type exprSandbox struct {
	expr Expr

	exprStack []Expr
	opStack   []*Token
}

func (esb *exprSandbox) build(stream *TokenStream) error {
	if stream.Size() == 0 {
		return errors.New("empty stream")
	}
	var (
		topOp *Token
		token *Token
		err   error
	)
	for stream.HasNext() {
		if token, err = stream.Next(); err != nil {
			return err
		}
		switch token.Type() {
		case TYPE_NUMBER, TYPE_STRING:
			b := &BasicLit{Kind: token.Type(), Value: token}
			esb.exprStack = append(esb.exprStack, b)

		case TYPE_NAME:
			if strings.Contains(internalKeyWords, fmt.Sprintf("_%s_", token.value)) {
				return newUnexpectedToken(token)
			}
			if !stream.IsEOF() {
				if nextToken, err := stream.Peek(1); err == nil && nextToken.value == "(" {
					esb.exprStack = append(esb.exprStack, &CallExpr{Func: &Ident{Name: token}})
					continue
				}
			}
			esb.exprStack = append(esb.exprStack, &Ident{Name: token})

		case TYPE_OPERATOR:
			switch token.value {
			case ")":
				for {
					topOp = esb.opStack[len(esb.opStack)-1]
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if topOp.value != "(" {
						esb.mergeExprStack(topOp)
						continue
					}
					if topOp.value == "(" || len(esb.opStack) == 0 {
						break
					}
				}

			case "]":
				for {
					topOp = esb.opStack[len(esb.opStack)-1]
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					esb.mergeExprStack(topOp)
					if topOp.value == "[" || len(esb.opStack) == 0 {
						break
					}
				}

			case "(":
				esb.opStack = append(esb.opStack, token)

			case "[":
				if len(esb.opStack) == 0 {
					esb.opStack = append(esb.opStack, token)
					continue
				}
				for {
					topOp = esb.opStack[len(esb.opStack)-1]
					if topOp.value != "." {
						break
					}
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
				}
				esb.opStack = append(esb.opStack, token)

			default:
				if !allowOp(token) {
					return newUnexpectedToken(token)
				}
				if len(esb.opStack) == 0 {
					esb.opStack = append(esb.opStack, token)
					continue
				}
				topOp = esb.opStack[len(esb.opStack)-1]
				for compare(topOp.value, token.value) {
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
			switch token.value {
			case ",":
				topOp = esb.opStack[len(esb.opStack)-1]
				for topOp.value != "(" {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.mergeExprStack(token)

			default:
				return newUnexpectedToken(token)

			}
		case TYPE_EOF:
		default:
			return newUnexpectedToken(token)
		}
	}

	for i := len(esb.opStack) - 1; i >= 0; i-- {
		topOp = esb.opStack[i]
		esb.opStack = esb.opStack[:i]
		if err = esb.mergeExprStack(topOp); err != nil {
			return err
		}
	}

	if len(esb.exprStack) != 1 {
		return errors.Errorf("parse expr failed1: %s", stream.String())
	}
	esb.expr = esb.exprStack[0]

	return nil
}

func (esb *exprSandbox) reset() {
	esb.expr = nil
	esb.exprStack = esb.exprStack[0:0]
	esb.opStack = esb.opStack[0:0]
}

func (esb *exprSandbox) mergeExprStack(token *Token) error {
	if len(esb.exprStack) < 2 {
		return newUnexpectedToken(token)
	}
	var expr1, expr2 = esb.exprStack[len(esb.exprStack)-1], esb.exprStack[len(esb.exprStack)-2]
	switch token.value {
	case "+", "-", "*", "/", ">", "==", "<", ">=", "<=", "and", "or":
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		esb.exprStack = append(esb.exprStack, &BinaryExpr{X: expr2, Op: token, Y: expr1})

	case "not", "++", "--":
		esb.exprStack[len(esb.exprStack)-1] = &SingleExpr{X: expr1, Op: token}

	case "[", ".":
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		esb.exprStack = append(esb.exprStack, &IndexExpr{X: expr2, Op: token, Index: expr1})

	case ",":
		if list, ok := expr2.(*ListExpr); ok {
			list.List = append(list.List, expr1)
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-1]
		} else {
			list := &ListExpr{}
			list.List = append(list.List, expr1)
			esb.exprStack[len(esb.exprStack)-1] = list
		}

	case ")":
		if list, ok := expr2.(*ListExpr); ok {
			if len(esb.exprStack) < 3 {
				return errors.Errorf("Unexpected arg list %s", list.Literal())
			}
			list.List = append(list.List, expr1)
			expr3 := esb.exprStack[len(esb.exprStack)-3]
			if fn, ok := expr3.(*CallExpr); !ok {
				return errors.Errorf("Unexpected tokens %s", expr3.Literal())
			} else {
				fn.Args = list
			}
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		} else if fn, ok := expr2.(*CallExpr); ok {
			list := &ListExpr{}
			list.List = append(list.List, expr1)
			fn.Args = list
			esb.exprStack = esb.exprStack[:len(esb.exprStack)-1]
		} else {
			return errors.Errorf("Unexpected tokens %s", expr2.Literal())
		}

	default:
		return newUnexpectedToken(token)

	}

	return nil
}

func subStreamIf(ts *TokenStream, fn func(t *Token) bool) (*TokenStream, error) {
	if _, err := ts.Skip(1); err != nil {
		return nil, err
	}
	start := ts.current
	for ts.HasNext() {
		if token, err := ts.Next(); err != nil {
			return nil, err
		} else if !fn(token) {
			break
		}
	}
	if start == ts.current {
		if token, err := ts.Current(); err != nil {
			return nil, err
		} else {
			return nil, newUnexpectedToken(token)
		}
	}
	le := ts.current - start
	var tokens = make([]*Token, le+1)
	copy(tokens, ts.tokens[start:ts.current])
	tokens[le] = newToken(TYPE_EOF, "", ts.tokens[ts.current-1].Line())

	return &TokenStream{Source: ts.Source, tokens: tokens, current: -1}, nil
}

func nextTokenValueShouldBe(ts *TokenStream, value string) (*Token, error) {
	if token, err := ts.Next(); err != nil {
		return nil, err
	} else if token.value != value {
		return nil, newUnexpectedToken(token)
	} else {
		return token, nil
	}
}

func nextTokenTypeShouldBe(ts *TokenStream, typ int) (*Token, error) {
	var (
		token *Token
		err   error
	)
	if token, err = ts.Next(); err != nil {
		return nil, err
	} else if token.typ != typ {
		return nil, newUnexpectedToken(token)
	} else {
		return token, nil
	}
}
