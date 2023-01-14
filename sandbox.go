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

func buildTemplate(content string) (*Document, error) {
	var (
		source *sourceCode
		err    error
	)
	source = newSourceCode(content)
	if err != nil {
		return nil, err
	}

	return buildSource(source)
}

func buildFileTemplate(path string) (doc *Document, err error) {
	if doc = _cache.Doc(path); doc != nil {
		return
	}
	var source *sourceCode
	source, err = newSourceCodeFile(path)
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

func buildSource(source *sourceCode) (*Document, error) {
	var (
		stream *tokenStream
		err    error
	)
	if err != nil {
		return nil, err
	}
	stream, err = tokenize(source)
	if err != nil {
		return nil, err
	}
	doc := NewDocument()
	err = build(doc, stream)

	return doc, err
}

func build(doc *Document, stream *tokenStream) error {
	sb := getSandbox()
	defer putSandbox(sb)
	err := sb.build(doc, stream)

	return err
}

func compare(op1, op2 string) bool {
	return rank[op1] <= rank[op2]
}

func allowOp(op *token) bool {
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

func (sb *sandbox) build(doc *Document, stream *tokenStream) error {
	sb.cursor = doc
	var (
		tok       *token
		err       error
		node      Direct
		subStream *tokenStream
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

	for stream.hasNext() {
		if tok, err = stream.next(); err != nil {
			return err
		}

		switch tok.typ {
		case type_eof:
			return nil

		case type_text:
			node = &TextDirect{Text: &BasicLit{Kind: type_string, Value: tok}}
			sb.cursor.Append(node)

		case type_var_start:
			if subStream, err = subStreamIf(stream, func(t *token) bool {
				return t.typ != type_var_end
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

		case type_command_start:
			tok, err = stream.next()
			if err != nil {
				return err
			}
			switch tok.value {
			case "endblock":
				if _, ok = sb.cursor.(*BlockDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popStack()

			case "endif":
				if _, ok = sb.cursor.(*IfDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popStack()

			case "endfor":
				if _, ok = sb.cursor.(*ForDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popStack()

			case "extend":
				node = &ExtendDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_string); err != nil {
					return err
				}
				node.(*ExtendDirect).Path = &BasicLit{Kind: tok.typ, Value: tok}
				if baseDoc, err := buildFileTemplate(tok.value); err != nil {
					return err
				} else {
					baseDoc.extended = true
					node.(*ExtendDirect).Doc = baseDoc
				}
				doc.Extend = node.(*ExtendDirect)

			case "include":
				node = &IncludeDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_string); err != nil {
					return err
				}
				node.(*IncludeDirect).Path = &BasicLit{Kind: tok.typ, Value: tok}
				if baseDoc, err = buildFileTemplate(tok.value); err != nil {
					return err
				} else {
					node.(*IncludeDirect).Doc = baseDoc
				}
				if tok, err = stream.next(); err != nil {
					return err
				}
				if tok.value == "with" {
					if subStream, err = subStreamIf(stream, func(t *token) bool {
						return t.typ != type_command_end && t.value != "only"
					}); err != nil {
						return err
					}
					box = getExprSandbox()
					boxes = append(boxes, box)
					if err = box.build(subStream); err != nil {
						return err
					}
					node.(*IncludeDirect).Params = box.expr

					if token, err := stream.current(); err != nil {
						return err
					} else if token.value == "only" {
						node.(*IncludeDirect).Only = true
					}
				}
				sb.cursor.Append(node)

			case "block":
				if tok, err = nextTokenTypeShouldBe(stream, type_name); err != nil {
					return err
				}
				node = &BlockDirect{Name: &BasicLit{Kind: type_string, Value: tok}}
				if _, ok := doc.blocks[tok.value]; ok {
					return errors.Errorf("block %s has already exist", tok.value)
				}
				doc.blocks[tok.value] = node.(*BlockDirect)
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node.(*BlockDirect))

			case "set":
				node = &AssignDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_name); err != nil {
					return err
				}
				node.(*AssignDirect).Lh = &Ident{Name: tok}
				if _, err = nextTokenValueShouldBe(stream, "="); err != nil {
					return err
				}
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.typ != type_command_end
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
					return newUnexpectedToken(tok)
				}
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.typ != type_command_end
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
					return newUnexpectedToken(tok)
				}
				sb.cursor.(*IfDirect).Else = &SectionDirect{}

			case "if":
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.typ != type_command_end
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
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.value != "in"
				}); err != nil {
					return err
				}
				node = &ForDirect{}
				switch subStream.size() {
				case 2:
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					node.(*ForDirect).Value = &Ident{Name: tok}
				case 4:
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					node.(*ForDirect).Key = &Ident{Name: tok}
					subStream.skip(1)
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					node.(*ForDirect).Value = &Ident{Name: tok}
				default:
					return errors.Errorf("Unexpected arg list %s in for loop", subStream.string())
				}

				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.typ != type_command_end
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
				return newUnexpectedToken(tok)

			}

		case type_command_end, type_var_end:
			continue

		default:
			return newUnexpectedToken(tok)

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
	opStack   []*token
}

func (esb *exprSandbox) build(stream *tokenStream) error {
	if stream.size() == 0 {
		return errors.New("empty stream")
	}
	var (
		topOp *token
		tok   *token
		err   error
	)
	for stream.hasNext() {
		if tok, err = stream.next(); err != nil {
			return err
		}
		switch tok.typ {
		case type_number, type_string:
			b := &BasicLit{Kind: tok.typ, Value: tok}
			esb.exprStack = append(esb.exprStack, b)

		case type_name:
			if strings.Contains(internalKeyWords, fmt.Sprintf("_%s_", tok.value)) {
				return newUnexpectedToken(tok)
			}
			if stream.hasNext() {
				if nextToken, err := stream.peek(1); err == nil && nextToken.value == "(" {
					esb.exprStack = append(esb.exprStack, &CallExpr{Func: &Ident{Name: tok}})
					continue
				}
			}
			esb.exprStack = append(esb.exprStack, &Ident{Name: tok})

		case type_operator:
			switch tok.value {
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
				esb.opStack = append(esb.opStack, tok)

			case "[":
				if len(esb.opStack) == 0 {
					esb.opStack = append(esb.opStack, tok)
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
				esb.opStack = append(esb.opStack, tok)

			default:
				if !allowOp(tok) {
					return newUnexpectedToken(tok)
				}
				if len(esb.opStack) == 0 {
					esb.opStack = append(esb.opStack, tok)
					continue
				}
				topOp = esb.opStack[len(esb.opStack)-1]
				for compare(topOp.value, tok.value) {
					esb.mergeExprStack(topOp)
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if len(esb.opStack) == 0 {
						break
					}
					topOp = esb.opStack[len(esb.opStack)-1]
				}
				esb.opStack = append(esb.opStack, tok)

			}

		case type_punctuation:
			switch tok.value {
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
				esb.mergeExprStack(tok)

			default:
				return newUnexpectedToken(tok)

			}
		case type_eof:
		default:
			return newUnexpectedToken(tok)
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
		return errors.Errorf("parse expr failed1: %s", stream.string())
	}
	esb.expr = esb.exprStack[0]

	return nil
}

func (esb *exprSandbox) reset() {
	esb.expr = nil
	esb.exprStack = esb.exprStack[0:0]
	esb.opStack = esb.opStack[0:0]
}

func (esb *exprSandbox) mergeExprStack(tok *token) error {
	if len(esb.exprStack) < 2 {
		return newUnexpectedToken(tok)
	}
	var expr1, expr2 = esb.exprStack[len(esb.exprStack)-1], esb.exprStack[len(esb.exprStack)-2]
	switch tok.value {
	case "+", "-", "*", "/", ">", "==", "<", ">=", "<=", "and", "or":
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		esb.exprStack = append(esb.exprStack, &BinaryExpr{X: expr2, Op: tok, Y: expr1})

	case "not", "++", "--":
		esb.exprStack[len(esb.exprStack)-1] = &SingleExpr{X: expr1, Op: tok}

	case "[", ".":
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		esb.exprStack = append(esb.exprStack, &IndexExpr{X: expr2, Op: tok, Index: expr1})

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
		return newUnexpectedToken(tok)

	}

	return nil
}

func nextTokenValueShouldBe(ts *tokenStream, value string) (tok *token, err error) {
	if tok, err = ts.next(); err != nil {
		return nil, err
	} else if tok.value != value {
		return nil, newUnexpectedToken(tok)
	} else {
		return tok, nil
	}
}

func nextTokenTypeShouldBe(ts *tokenStream, typ int) (tok *token, err error) {
	if tok, err = ts.next(); err != nil {
		return nil, err
	} else if tok.typ != typ {
		return nil, newUnexpectedToken(tok)
	} else {
		return tok, nil
	}
}
