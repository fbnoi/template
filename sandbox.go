package template

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	rank = map[string]int{
		"(":   0,
		")":   0,
		".":   1,
		"[":   2,
		"]":   2,
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
	sb := GetSandbox()
	defer PutSandbox(sb)
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

func GetSandbox() *sandbox {
	return sandboxPool.Get().(*sandbox)
}

func PutSandbox(sb *sandbox) {
	sb.reset()
	sandboxPool.Put(sb)
}

func GetExprSandbox() *exprSandbox {
	return exprSandboxPool.Get().(*exprSandbox)
}

func PutExprSandbox(sb *exprSandbox) {
	sb.reset()
	exprSandboxPool.Put(sb)
}

func parseJsonParams(stream *TokenStream) (string, error) {
	sb := &strings.Builder{}
	bracketLock := 0
	for !stream.IsEOF() {
		token, err := stream.Next()
		if err != nil {
			return "", err
		}
		sb.WriteString(token.value)
		if token.value == ":" {
			if pt, err := stream.Peek(1); err == nil && (pt.typ == TYPE_NAME || pt.value == "(") {
				sb.WriteString("\"")
			}
		}

		if token.typ == TYPE_NAME || token.value == ")]" {
			if pt, err := stream.Peek(1); err == nil && (pt.value == "," || pt.value == "}") && bracketLock == 0 {
				sb.WriteString("\"")
			}
		}

		if strings.Contains("([", token.value) {
			bracketLock++
		} else if strings.Contains(")]", token.value) {
			bracketLock--
		}
	}

	return sb.String(), nil
}

type sandbox struct {
	cursor AppendAble
	stack  []AppendAble
}

func (sb *sandbox) build(doc *Document, stream *TokenStream) error {
	sb.cursor = doc
	for stream.HasNext() {
		token, _ := stream.Next()
		switch token.Type() {
		case TYPE_EOF:
			return nil

		case TYPE_TEXT:
			node := &TextDirect{Text: &BasicLit{Kind: TYPE_STRING, Value: token}}
			sb.cursor.Append(node)

		case TYPE_VAR_START:
			i := stream.CurrentIndex() + 1
			for token.Type() != TYPE_VAR_END {
				token, _ = stream.Next()
			}
			if i >= stream.CurrentIndex() {
				return newUnexpectedToken(token)
			}
			subStream := stream.SubStream(i, stream.CurrentIndex())
			box := GetExprSandbox()
			defer PutExprSandbox(box)
			if err := box.build(subStream); err != nil {
				return err
			}
			node := &ValueDirect{Tok: box.expr}
			sb.cursor.Append(node)

		case TYPE_COMMAND_START:
			token, _ = stream.Next()
			switch token.value {
			case "endblock":
				if _, ok := sb.cursor.(*BlockDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "endif":
				if _, ok := sb.cursor.(*IfDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "endfor":
				if _, ok := sb.cursor.(*ForDirect); !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor = sb.popStack()

			case "extend":
				node := &ExtendDirect{}
				token, _ := stream.Next()
				if token.Type() != TYPE_STRING {
					return newUnexpectedToken(token)
				}
				node.Path = &BasicLit{Kind: token.Type(), Value: token}
				if baseDoc, err := BuildFileTemplate(token.value); err != nil {
					return err
				} else {
					baseDoc.extended = true
					node.Doc = baseDoc
				}
				doc.Extend = node
				token, _ = stream.Next()
				if token.Type() != TYPE_COMMAND_END {
					return newUnexpectedToken(token)
				}

			case "include":
				node := &IncludeDirect{}
				token, _ := stream.Next()
				if token.Type() != TYPE_STRING {
					return newUnexpectedToken(token)
				}
				node.Path = &BasicLit{Kind: token.Type(), Value: token}
				if baseDoc, err := BuildFileTemplate(node.Path.Value.value); err != nil {
					return err
				} else {
					node.Doc = baseDoc
				}
				token, _ = stream.Next()
				if token.value == "with" {
					token, _ = stream.Next()
					i := stream.CurrentIndex()
					for token.Type() != TYPE_COMMAND_END {
						token, _ = stream.Next()
					}
					if i >= stream.CurrentIndex() {
						return newUnexpectedToken(token)
					}
					subStream := stream.SubStream(i, stream.CurrentIndex())
					str, err := parseJsonParams(subStream)
					if err != nil {
						return err
					}
					ps := Params{}
					if err := json.Unmarshal([]byte(str), &ps); err != nil {
						return err
					}
					if token, err := stream.Next(); err != nil {
						return err
					} else if token.Value() == "only" {
						node.Only = true
						if token, err := stream.Next(); err != nil {
							return err
						} else if token.typ != TYPE_COMMAND_END {
							return newUnexpectedToken(token)
						}
					} else if token.typ != TYPE_COMMAND_END {
						return newUnexpectedToken(token)
					}
				} else if token.typ != TYPE_COMMAND_END {
					return newUnexpectedToken(token)
				}
				sb.cursor.Append(node)

			case "block":
				token, _ := stream.Next()
				if token.Type() != TYPE_NAME {
					return newUnexpectedToken(token)
				}
				node := &BlockDirect{Name: &BasicLit{Kind: TYPE_STRING, Value: token}}
				if _, ok := doc.blocks[token.value]; ok {
					return errors.Errorf("block %s has already exist", token.value)
				}
				doc.blocks[token.value] = node
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node)

			case "set":
				node := &AssignDirect{}
				token, _ = stream.Next()
				if token.Type() != TYPE_NAME {
					return newUnexpectedToken(token)
				}
				node.Lh = &Ident{Name: token}

				token, _ = stream.Next()
				if token.value != "=" {
					return newUnexpectedToken(token)
				}

				token, _ = stream.Next()
				i := stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = stream.Next()
				}
				if i >= stream.CurrentIndex() {
					return newUnexpectedToken(token)
				}
				subStream := stream.SubStream(i, stream.CurrentIndex())
				box := GetExprSandbox()
				defer PutExprSandbox(box)
				if err := box.build(subStream); err != nil {
					return err
				}
				node.Rh = box.expr
				sb.cursor.Append(node)

			case "elseif":
				ifNode, ok := sb.cursor.(*IfDirect)
				if !ok {
					return newUnexpectedToken(token)
				}
				node := &IfDirect{}
				token, _ = stream.Next()
				i := stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = stream.Next()
				}
				if i >= stream.CurrentIndex() {
					return newUnexpectedToken(token)
				}
				subStream := stream.SubStream(i, stream.CurrentIndex())
				box := GetExprSandbox()
				defer PutExprSandbox(box)
				if err := box.build(subStream); err != nil {
					return err
				}
				node.Cond = box.expr
				ifNode.Else = node
				sb.shiftStack(node)

			case "else":
				_, ok := sb.cursor.(*IfDirect)
				if !ok {
					return newUnexpectedToken(token)
				}
				sb.cursor.(*IfDirect).Else = &SectionDirect{}

			case "if": // if condition
				node := &IfDirect{}
				token, _ = stream.Next()
				i := stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = stream.Next()
				}
				if i >= stream.CurrentIndex() {
					return newUnexpectedToken(token)
				}
				subStream := stream.SubStream(i, stream.CurrentIndex())
				box := GetExprSandbox()
				defer PutExprSandbox(box)
				if err := box.build(subStream); err != nil {
					return err
				}
				node.Cond = box.expr
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node)

			case "for": // for key, value in list, for value in list, for key, _ in list
				node := &ForDirect{}
				kToken, err := stream.Next()
				if err != nil {
					return err
				}
				if kToken.Type() != TYPE_NAME {
					return newUnexpectedToken(kToken)
				}
				vToken, err := stream.Next()
				if err != nil {
					return err
				}
				if vToken.value == "in" {
					node.Value = &Ident{Name: vToken}
				} else if vToken.value == "," {
					if vToken, err = stream.Next(); err != nil {
						return err
					}
					node.Key, node.Value = &Ident{Name: kToken}, &Ident{Name: vToken}
					if nToken, err := stream.Next(); err != nil {
						return err
					} else if nToken.value != "in" {
						return newUnexpectedToken(nToken)
					}
				} else {
					return newUnexpectedToken(vToken)
				}
				token, _ = stream.Next()
				i := stream.CurrentIndex()
				for token.Type() != TYPE_COMMAND_END {
					token, _ = stream.Next()
				}
				if i >= stream.CurrentIndex() {
					return newUnexpectedToken(token)
				}
				subStream := stream.SubStream(i, stream.CurrentIndex())
				box := GetExprSandbox()
				defer PutExprSandbox(box)
				if err := box.build(subStream); err != nil {
					return err
				}
				node.X = box.expr
				sb.cursor.Append(node)
				sb.cursor = sb.pushStack(node)

			default:
				return newUnexpectedToken(token)

			}
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
	for stream.HasNext() {
		token, _ := stream.Next()
		switch token.Type() {
		case TYPE_NUMBER, TYPE_STRING:
			b := &BasicLit{Kind: token.Type(), Value: token}
			esb.exprStack = append(esb.exprStack, b)

		case TYPE_NAME:
			if strings.Contains(internalKeyWords, fmt.Sprintf("_%s_", token.value)) {
				return newUnexpectedToken(token)
			}
			i := &Ident{Name: token}
			if !stream.IsEOF() {
				if nextToken, err := stream.Peek(1); err == nil && nextToken.value == "(" {
					c := &CallExpr{Func: i}
					esb.exprStack = append(esb.exprStack, c)
					continue
				}
			}
			esb.exprStack = append(esb.exprStack, i)

		case TYPE_OPERATOR:
			if token.value == ")" {
				for {
					topOp := esb.opStack[len(esb.opStack)-1]
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					if topOp.value != "(" {
						esb.mergeExprStack(topOp)
						continue
					}
					if topOp.value == "(" || len(esb.opStack) == 0 {
						break
					}
				}
			} else if token.value == "]" {
				for {
					topOp := esb.opStack[len(esb.opStack)-1]
					esb.opStack = esb.opStack[:len(esb.opStack)-1]
					esb.mergeExprStack(topOp)
					if topOp.value == "[" || len(esb.opStack) == 0 {
						break
					}
				}
			} else {
				if !allowOp(token) {
					return newUnexpectedToken(token)
				}
				if len(esb.opStack) == 0 {
					esb.opStack = append(esb.opStack, token)
				} else {
					topOp := esb.opStack[len(esb.opStack)-1]
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
			}

		case TYPE_PUNCTUATION:
			switch token.value {
			case ",":
				topOp := esb.opStack[len(esb.opStack)-1]
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
		topOp := esb.opStack[i]
		esb.opStack = esb.opStack[:i]
		if err := esb.mergeExprStack(topOp); err != nil {
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
	switch token.value {
	case "+", "-", "*", "/", ">", "==", "<", ">=", "<=", "and", "or":
		expr1 := esb.exprStack[len(esb.exprStack)-2]
		expr2 := esb.exprStack[len(esb.exprStack)-1]
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		b := &BinaryExpr{X: expr1, Op: token, Y: expr2}
		esb.exprStack = append(esb.exprStack, b)

	case "not", "++", "--":
		expr1 := esb.exprStack[len(esb.exprStack)-1]
		expr1 = &SingleExpr{X: expr1, Op: token}

	case "[", ".":
		expr1 := esb.exprStack[len(esb.exprStack)-2]
		expr2 := esb.exprStack[len(esb.exprStack)-1]
		esb.exprStack = esb.exprStack[:len(esb.exprStack)-2]
		i := &IndexExpr{X: expr1, Op: token, Index: expr2}
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
		return newUnexpectedToken(token)

	}

	return nil
}
