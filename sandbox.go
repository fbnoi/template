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
		"|":   2,
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
	source := newSourceCode(content)

	return buildSource(source)
}

func buildFileTemplate(path string) (doc *Document, err error) {
	var source *sourceCode
	source, err = newSourceCodeFile(path)
	if err != nil {
		return nil, err
	}
	doc, err = buildSource(source)

	return
}

func buildSource(source *sourceCode) (*Document, error) {
	if doc := _cache.doc(source.identity); doc != nil {
		return doc, nil
	}

	if stream, err := tokenize(source); err != nil {
		return nil, err
	} else {
		doc := newDocument()
		err = build(doc, stream)
		_cache.addDoc(source.identity, doc)

		return doc, err
	}
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
	cursor appendAble
	stack  []appendAble
}

func (sb *sandbox) build(doc *Document, stream *tokenStream) error {
	sb.cursor = doc
	var (
		tok       *token
		err       error
		node      direct
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
		case type_text:
			node = &textDirect{text: &basicLit{kind: type_string, value: tok}}
			sb.cursor.append(node)

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
				node = &valueDirect{tok: box.expr}
				sb.cursor.append(node)
			}

		case type_command_start:
			tok, err = stream.next()
			if err != nil {
				return err
			}
			switch tok.value {
			case "endblock":
				if _, ok = sb.cursor.(*blockDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popsStack()

			case "endif":
				if _, ok = sb.cursor.(*ifDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popsStack()

			case "endfor":
				if _, ok = sb.cursor.(*forDirect); !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor = sb.popsStack()

			case "extend":
				node = &extendDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_string); err != nil {
					return err
				}
				node.(*extendDirect).path = &basicLit{kind: tok.typ, value: tok}
				if baseDoc, err := buildFileTemplate(tok.value); err != nil {
					return err
				} else {
					baseDoc.extended = true
					node.(*extendDirect).doc = baseDoc
				}
				doc.extend = node.(*extendDirect)

			case "include":
				node = &includeDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_string); err != nil {
					return err
				}
				node.(*includeDirect).path = &basicLit{kind: tok.typ, value: tok}
				if baseDoc, err = buildFileTemplate(tok.value); err != nil {
					return err
				} else {
					node.(*includeDirect).doc = baseDoc
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
					node.(*includeDirect).params = box.expr

					if token, err := stream.current(); err != nil {
						return err
					} else if token.value == "only" {
						node.(*includeDirect).only = true
					}
				}
				sb.cursor.append(node)

			case "block":
				if tok, err = nextTokenTypeShouldBe(stream, type_name); err != nil {
					return err
				}
				node = &blockDirect{name: &basicLit{kind: type_string, value: tok}}
				if _, ok := doc.blocks[tok.value]; ok {
					return errors.Errorf("block %s has already exist", tok.value)
				}
				doc.blocks[tok.value] = node.(*blockDirect)
				sb.cursor.append(node)
				sb.cursor = sb.pushStack(node.(*blockDirect))

			case "set":
				node = &assignDirect{}
				if tok, err = nextTokenTypeShouldBe(stream, type_name); err != nil {
					return err
				}
				node.(*assignDirect).lh = &ident{name: tok}
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
				node.(*assignDirect).rh = box.expr
				sb.cursor.append(node)

			case "elseif":
				ifNode, ok := sb.cursor.(*ifDirect)
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
				node = &ifDirect{}
				node.(*ifDirect).cond = box.expr
				ifNode.el = node
				sb.shiftStack(node.(*ifDirect))

			case "else":
				_, ok := sb.cursor.(*ifDirect)
				if !ok {
					return newUnexpectedToken(tok)
				}
				sb.cursor.(*ifDirect).el = &sectionDirect{}

			case "if":
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.typ != type_command_end
				}); err != nil {
					return err
				}
				node = &ifDirect{}
				box = getExprSandbox()
				boxes = append(boxes, box)
				if err = box.build(subStream); err != nil {
					return err
				}
				node.(*ifDirect).cond = box.expr
				sb.cursor.append(node)
				sb.cursor = sb.pushStack(node.(*ifDirect))

			case "for":
				if subStream, err = subStreamIf(stream, func(t *token) bool {
					return t.value != "in"
				}); err != nil {
					return err
				}
				node = &forDirect{}
				switch subStream.size() {
				case 1:
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					node.(*forDirect).value = &ident{name: tok}
				case 3:
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					if tok.value != "_" {
						node.(*forDirect).key = &ident{name: tok}
					}
					subStream.skip(1)
					if tok, err = nextTokenTypeShouldBe(subStream, type_name); err != nil {
						return err
					}
					node.(*forDirect).value = &ident{name: tok}
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
				node.(*forDirect).x = box.expr
				sb.cursor.append(node)
				sb.cursor = sb.pushStack(node.(*forDirect))

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

func (sb *sandbox) pushStack(node appendAble) appendAble {
	sb.stack = append(sb.stack, sb.cursor)

	return node
}

func (sb *sandbox) popsStack() appendAble {
	tmp := sb.stack[len(sb.stack)-1]
	sb.stack = sb.stack[:len(sb.stack)-1]

	return tmp
}

func (sb *sandbox) shiftStack(node appendAble) {
	sb.cursor = node
}

func (sb *sandbox) reset() {
	sb.cursor = nil
	sb.stack = sb.stack[0:0]
}

type exprSandbox struct {
	expr       expr
	exprsStack []expr
	opsStack   []*token
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
		case type_number, type_string, type_bool:
			b := &basicLit{kind: tok.typ, value: tok}
			esb.exprsStack = append(esb.exprsStack, b)

		case type_name:
			if strings.Contains(internalKeyWords, fmt.Sprintf("_%s_", tok.value)) {
				return newUnexpectedToken(tok)
			}
			if stream.hasNext() {
				if nextToken, err := stream.peek(1); err == nil && nextToken.value == "(" {
					esb.exprsStack = append(esb.exprsStack, &callExpr{fn: &ident{name: tok}})
					continue
				}
			}
			esb.exprsStack = append(esb.exprsStack, &ident{name: tok})

		case type_operator:
			switch tok.value {
			case ")", "]":
				for {
					topOp = esb.opsStack[len(esb.opsStack)-1]
					esb.opsStack = esb.opsStack[:len(esb.opsStack)-1]
					esb.mergeExprsStack(topOp)
					if topOp.value == "(" || topOp.value == "[" || len(esb.opsStack) == 0 {
						break
					}
				}

			case "(":
				esb.opsStack = append(esb.opsStack, tok)

			case "[":
				if len(esb.opsStack) == 0 {
					esb.opsStack = append(esb.opsStack, tok)
					continue
				}
				for {
					topOp = esb.opsStack[len(esb.opsStack)-1]
					if topOp.value != "." {
						break
					}
					esb.mergeExprsStack(topOp)
					esb.opsStack = esb.opsStack[:len(esb.opsStack)-1]
				}
				esb.opsStack = append(esb.opsStack, tok)

			default:
				if !allowOp(tok) {
					return newUnexpectedToken(tok)
				}
				if len(esb.opsStack) == 0 {
					esb.opsStack = append(esb.opsStack, tok)
					continue
				}
				topOp = esb.opsStack[len(esb.opsStack)-1]
				for compare(topOp.value, tok.value) {
					esb.mergeExprsStack(topOp)
					esb.opsStack = esb.opsStack[:len(esb.opsStack)-1]
					if len(esb.opsStack) == 0 {
						break
					}
					topOp = esb.opsStack[len(esb.opsStack)-1]
				}
				esb.opsStack = append(esb.opsStack, tok)

			}

		case type_punctuation:
			switch tok.value {
			case ",":
				topOp = esb.opsStack[len(esb.opsStack)-1]
				for topOp.value != "(" {
					esb.mergeExprsStack(topOp)
					esb.opsStack = esb.opsStack[:len(esb.opsStack)-1]
					if len(esb.opsStack) == 0 {
						break
					}
					topOp = esb.opsStack[len(esb.opsStack)-1]
				}
				esb.opsStack = append(esb.opsStack, tok)

			default:
				return newUnexpectedToken(tok)

			}
		default:
			return newUnexpectedToken(tok)
		}
	}

	for i := len(esb.opsStack) - 1; i >= 0; i-- {
		topOp = esb.opsStack[i]
		esb.opsStack = esb.opsStack[:i]
		if err = esb.mergeExprsStack(topOp); err != nil {
			return err
		}
	}

	if len(esb.exprsStack) != 1 {
		return errors.Errorf("parse expr failed1: %s", stream.string())
	}
	esb.expr = esb.exprsStack[0]

	return nil
}

func (esb *exprSandbox) reset() {
	esb.expr = nil
	esb.exprsStack = esb.exprsStack[0:0]
	esb.opsStack = esb.opsStack[0:0]
}

func (esb *exprSandbox) mergeExprsStack(tok *token) error {
	if len(esb.exprsStack) < 2 {
		return newUnexpectedToken(tok)
	}
	var expr1, expr2 = esb.exprsStack[len(esb.exprsStack)-1], esb.exprsStack[len(esb.exprsStack)-2]
	switch tok.value {
	case "+", "-", "*", "/", ">", "==", "<", ">=", "<=", "and", "or", "|":
		esb.exprsStack = esb.exprsStack[:len(esb.exprsStack)-2]
		esb.exprsStack = append(esb.exprsStack, &binaryExpr{x: expr2, op: tok, y: expr1})

	case "not", "++", "--":
		esb.exprsStack[len(esb.exprsStack)-1] = &singleExpr{x: expr1, op: tok}

	case "[", ".":
		esb.exprsStack = esb.exprsStack[:len(esb.exprsStack)-2]
		esb.exprsStack = append(esb.exprsStack, &indexExpr{x: expr2, op: tok, index: expr1})

	case ",":
		var (
			lExpr *listExpr
			ok    bool
		)
		if lExpr, ok = expr1.(*listExpr); ok {
			lExpr.list = append([]expr{expr2}, lExpr.list...)
		} else {
			lExpr = &listExpr{}
			lExpr.list = append(lExpr.list, expr1)
			lExpr.list = append([]expr{expr2}, lExpr.list...)
		}
		esb.exprsStack = esb.exprsStack[:len(esb.exprsStack)-2]
		esb.exprsStack = append(esb.exprsStack, lExpr)

	case "(":
		if fnExpr, ok := expr2.(*callExpr); ok {
			if lExpr, ok := expr1.(*listExpr); ok {
				fnExpr.args = lExpr

			} else {
				lExpr = &listExpr{}
				lExpr.list = append(lExpr.list, expr1)
				fnExpr.args = lExpr
			}
			esb.exprsStack = esb.exprsStack[:len(esb.exprsStack)-1]
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
