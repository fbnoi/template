package template

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	tag_comment         = [...]string{`{#`, `#}`}
	tag_block           = [...]string{`{%`, `%}`}
	tag_variable        = [...]string{`{{`, `}}`}
	tag_escape_comment  = [...]string{`@{#`, `#}`}
	tag_escape_block    = [...]string{`@{%`, `%}`}
	tag_escape_variable = [...]string{`@{{`, `}}`}

	word_operators = [...]string{"and", "or", "in"}
)

var (
	// }}
	reg_variable = regexp.MustCompile(fmt.Sprintf(`\s*%s`, tag_variable[1]))
	// %}
	reg_block = regexp.MustCompile(fmt.Sprintf(`\s*%s`, tag_block[1]))
	// #}
	reg_comment = regexp.MustCompile(fmt.Sprintf(`\s*%s`, tag_comment[1]))
	// {{ or {% or {#
	reg_token_start = regexp.MustCompile(fmt.Sprintf(`(@?%s|@?%s|@?%s)`, tag_variable[0], tag_block[0], tag_comment[0]))
	// \r\n \n
	reg_enter = regexp.MustCompile(`(\r\n|\n)`)
	// whitespace
	reg_whitespace = regexp.MustCompile(`^\s+`)
	// . + - * / > < = !
	reg_operator = regexp.MustCompile(`^[\!\.\+\-*\/><=]{1,3}`)
	// bracket [ ] ( ) {}
	reg_bracket       = regexp.MustCompile(`^[\[\]\(\)\{\}]`)
	reg_bracket_open  = regexp.MustCompile(`^[\[\(\{]$`)
	reg_bracket_close = regexp.MustCompile(`^[\]\)\}]$`)
	// word
	reg_word = regexp.MustCompile(`^[a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*`)
	// number
	reg_number      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?([Ee][\+\-][0-9]+)?`)
	reg_punctuation = regexp.MustCompile(`^[\?,:]`)

	// string
	reg_string = regexp.MustCompile(`^"([^"\\\\]*(?:\\\\.[^"\\\\]*)*)"|^'([^\'\\\\]*(?:\\\\.[^\'\\\\]*)*)'`)
)

func tokenize(source *sourceCode) (*tokenStream, error) {
	var (
		code            = reg_enter.ReplaceAllString(source.code, "\n")
		stream          = &tokenStream{source: source, cursor: -1}
		poss            = reg_token_start.FindAllStringIndex(code, -1)
		cursor          = 0
		line            = 0
		posIndex        = 0
		codeLen         = len(code)
		pos, ends, sPos []int
		bk, word        string
		bks             []*bracket
		end, length     int
		tok             *token
	)

	moveCursor := func(n int) {
		cursor = n
		line = len(reg_enter.FindAllString(code[:n], -1)) + 1
	}

	if len(poss) == 0 {
		tok = newToken(type_text, code[cursor:], line)
		stream.tokens = append(stream.tokens, tok)
		cursor = len(code)
	}
	for posIndex < len(poss) {
		pos = poss[posIndex]
		if pos[0] < cursor {
			posIndex++
			continue
		} else if pos[0] > cursor {
			tok = newToken(type_text, code[cursor:pos[0]], line)
			stream.tokens = append(stream.tokens, tok)
			moveCursor(pos[0])
		}
		var reg *regexp.Regexp
		switch code[pos[0]:pos[1]] {
		case tag_escape_comment[0]:
			moveCursor(pos[0] + 1)
			ends = reg_comment.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_escape_comment[0]}
			}
			tok = newToken(type_text, code[cursor:cursor+ends[1]], line)
			stream.tokens = append(stream.tokens, tok)
			moveCursor(cursor + ends[1])
		case tag_escape_block[0]:
			moveCursor(pos[0] + 1)
			ends = reg_block.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_escape_block[0]}
			}
			tok = newToken(type_text, code[cursor:cursor+ends[1]], line)
			stream.tokens = append(stream.tokens, tok)
			moveCursor(cursor + ends[1])
		case tag_escape_variable[0]:
			moveCursor(pos[0] + 1)
			ends = reg_variable.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_escape_variable[0]}
			}
			tok = newToken(type_text, code[cursor:cursor+ends[1]], line)
			stream.tokens = append(stream.tokens, tok)
			moveCursor(cursor + ends[1])
		case tag_comment[0]:
			ends = reg_comment.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_comment[0]}
			}
			tok = newToken(type_text, code[cursor:cursor+ends[1]], line)
			stream.tokens = append(stream.tokens, tok)
			moveCursor(cursor + ends[1])
		case tag_block[0]:
			reg = reg_block

		case tag_variable[0]:
			reg = reg_variable

		default:
			return nil, &UnexpectedToken{Line: line, token: code[pos[0]:pos[1]]}
		}

		if reg == reg_block {
			tok = newToken(type_command_start, code[cursor:cursor+2], line)
		} else {
			tok = newToken(type_var_start, code[cursor:cursor+2], line)
		}
		stream.tokens = append(stream.tokens, tok)
		moveCursor(cursor + 2)
		ends = reg.FindStringIndex(code[cursor:])
		if ends == nil {
			return nil, &UnClosedToken{Line: line, token: tag_block[0]}
		}
		length = ends[1] - ends[0]
		end = cursor + ends[0]

		for cursor < end {
			if sPos = reg_whitespace.FindStringIndex(code[cursor:end]); sPos != nil {
				moveCursor(cursor + sPos[1])
				continue
			}
			if sPos = reg_operator.FindStringIndex(code[cursor:end]); sPos != nil {
				tok = newToken(type_operator, code[cursor:cursor+sPos[1]], line)
				stream.tokens = append(stream.tokens, tok)
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_word.FindStringIndex(code[cursor:end]); sPos != nil {
				word = code[cursor : cursor+sPos[1]]
				moveCursor(cursor + sPos[1])
				if isWordOperator(word) {
					tok = newToken(type_operator, word, line)
					stream.tokens = append(stream.tokens, tok)
					continue
				}
				stream.tokens = append(stream.tokens, newToken(type_name, word, line))
			} else if sPos = reg_number.FindStringIndex(code[cursor:end]); sPos != nil {
				tok = newToken(type_number, code[cursor:cursor+sPos[1]], line)
				stream.tokens = append(stream.tokens, tok)
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_string.FindStringIndex(code[cursor:end]); sPos != nil {
				tok = newToken(type_string, code[cursor:cursor+sPos[1]], line)
				stream.tokens = append(stream.tokens, tok)
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_punctuation.FindStringIndex(code[cursor:end]); sPos != nil {
				tok = newToken(type_punctuation, code[cursor:cursor+sPos[1]], line)
				stream.tokens = append(stream.tokens, tok)
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_bracket.FindStringIndex(code[cursor:end]); sPos != nil {
				bk = code[cursor+sPos[0] : cursor+sPos[1]]
				if reg_bracket_open.MatchString(bk) {
					bks = append(bks, &bracket{ch: bk, line: line})
				} else if reg_bracket_close.MatchString(bk) {
					if len(bk) == 0 {
						return nil, &UnexpectedToken{Line: line, token: bk}
					}
					switch {
					case bks[len(bks)-1].ch == "(" && bk != ")":
						return nil, &UnexpectedToken{Line: line, token: bk}
					case bks[len(bks)-1].ch == "[" && bk != "]":
						return nil, &UnexpectedToken{Line: line, token: bk}
					case bks[len(bks)-1].ch == "{" && bk != "}":
						return nil, &UnexpectedToken{Line: line, token: bk}
					}
					bks = bks[:len(bks)-1]
				}
				tok = newToken(type_operator, bk, line)
				stream.tokens = append(stream.tokens, tok)
				moveCursor(cursor + sPos[1])
			} else {
				return nil, &UnexpectedToken{Line: line, token: code[cursor:end]}
			}
		}
		if len(bks) > 0 {
			return nil, &UnClosedToken{Line: bks[0].line, token: bks[0].ch}
		}
		moveCursor(end)
		if reg == reg_block {
			tok = newToken(type_command_end, code[cursor:cursor+length], line)
		} else {
			tok = newToken(type_var_end, code[cursor:cursor+length], line)
		}
		stream.tokens = append(stream.tokens, tok)
		moveCursor(cursor + length)

		posIndex++
	}

	if cursor < codeLen {
		tok = newToken(type_text, code[cursor:codeLen], line)
		stream.tokens = append(stream.tokens, tok)
		moveCursor(codeLen)
	}

	return stream, nil
}

func newToken(typ int, value string, line int) *token {
	return &token{typ: typ, value: value, line: line}
}

type bracket struct {
	ch   string
	line int
}

type tokenStream struct {
	source *sourceCode
	tokens []*token
	cursor int
}

func (ts *tokenStream) size() int {
	return len(ts.tokens)
}

func (ts *tokenStream) string() string {
	sb := &strings.Builder{}
	for _, t := range ts.tokens {
		sb.WriteString(t.string())
	}

	return sb.String()
}

func (ts *tokenStream) current() (*token, error) {
	if ts.cursor >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.cursor], nil
}

func (ts *tokenStream) hasNext() bool {
	size := len(ts.tokens)

	return ts.cursor < size-1
}

func (ts *tokenStream) next() (*token, error) {
	ts.cursor++
	if ts.cursor > len(ts.tokens)-1 {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.cursor], nil
}

func (ts *tokenStream) skip(n int) (*token, error) {
	ts.cursor += n
	if ts.cursor >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.cursor], nil
}

func (ts *tokenStream) peek(n int) (*token, error) {
	if ts.cursor+n >= len(ts.tokens)-1 {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.cursor+n], nil
}

func subStreamIf(ts *tokenStream, fn func(tok *token) bool) (*tokenStream, error) {
	if _, err := ts.skip(1); err != nil {
		return nil, err
	}
	start := ts.cursor
	var (
		tok *token
		err error
	)
	for ts.hasNext() {
		if tok, err = ts.next(); err != nil {
			return nil, err
		} else if !fn(tok) {
			break
		}
	}
	if start == ts.cursor {
		if tok, err = ts.current(); err != nil {
			return nil, err
		} else {
			return nil, newUnexpectedToken(tok)
		}
	}
	var tokens = make([]*token, ts.cursor-start)
	copy(tokens, ts.tokens[start:ts.cursor])

	return &tokenStream{source: ts.source, tokens: tokens, cursor: -1}, nil
}

func isWordOperator(word string) bool {
	for _, v := range word_operators {
		if v == word {
			return true
		}
	}

	return false
}
