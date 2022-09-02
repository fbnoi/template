package template

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
}

func (sb *SandBox) Parse(stream *TokenStream) *SandBox {
	sb.stream = stream
	sb.err = sb.parse()

	return sb
}
func (sb *SandBox) Error() error
func (sb *SandBox) Doc() *Document

func (sb *SandBox) parse() error {
	sb.cursor = sb.doc
	var i = 0
	for !sb.stream.IsEOF() {
		token, _ := sb.stream.Next()
		switch token.Type() {
		case TYPE_EOF:
			return nil
		case TYPE_TEXT:
			node := &TextDirect{Text: &BasicLit{Kind: TYPE_STRING}}
			sb.cursor.Append(node)
		case TYPE_VAR_START:
			var j = 0
			for token.Type() != TYPE_VAR_END {
				token, _ = sb.stream.Next()
				j++
			}
			subStream = sb.stream.SubStream(i, j)

			node := &TextDirect{Text: &BasicLit{Kind: TYPE_STRING}}
			sb.cursor.Append(node)
		}
		i++
	}
}

type ExprSandBox struct {
	stream *TokenStream
	err    error
	expr   Expr
}

func (esb *ExprSandBox) Parse(stream *TokenStream) *ExprSandBox
func (esb *ExprSandBox) Error() error
func (esb *ExprSandBox) BinaryExpr() *BinaryExpr
func (esb *ExprSandBox) CallExpr() *CallExpr
func (esb *ExprSandBox) IndexExpr() *IndexExpr
