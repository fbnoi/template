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
	// name
	reg_word = regexp.MustCompile(`^[a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*`)
	// number
	reg_number      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?([Ee][\+\-][0-9]+)?`)
	reg_punctuation = regexp.MustCompile(`^[\?,:]`)

	// string
	reg_string = regexp.MustCompile(`^"([^"\\\\]*(?:\\\\.[^"\\\\]*)*)"|^'([^\'\\\\]*(?:\\\\.[^\'\\\\]*)*)'`)
)

func Tokenize(source *sourceCode) (*TokenStream, error) {
	var (
		code            = reg_enter.ReplaceAllString(source.code, "\n")
		stream          = &TokenStream{Source: source, current: -1}
		poss            = reg_token_start.FindAllStringIndex(code, -1)
		cursor          = 0
		line            = 0
		posIndex        = 0
		codeLen         = len(code)
		pos, ends, sPos []int
		bracket, word   string
		brackets        []*Bracket
		end, length     int
		token           *token
	)

	moveCursor := func(n int) {
		cursor = n
		line = len(reg_enter.FindAllString(code[:n], -1)) + 1
	}

	if len(poss) == 0 {
		stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:], line))
		cursor = len(code)
	}
	for posIndex < len(poss) {
		pos = poss[posIndex]
		if pos[0] < cursor {
			posIndex++
			continue
		} else if pos[0] > cursor {
			stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:pos[0]], line))
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
			stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case tag_escape_block[0]:
			moveCursor(pos[0] + 1)
			ends = reg_block.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_escape_block[0]}
			}
			stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case tag_escape_variable[0]:
			moveCursor(pos[0] + 1)
			ends = reg_variable.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_escape_variable[0]}
			}
			stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case tag_comment[0]:
			ends = reg_comment.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: tag_comment[0]}
			}
			stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case tag_block[0]:
			reg = reg_block

		case tag_variable[0]:
			reg = reg_variable

		default:
			return nil, &UnexpectedToken{Line: line, token: code[pos[0]:pos[1]]}
		}

		if reg == reg_block {
			token = newToken(type_command_start, code[cursor:cursor+2], line)
		} else {
			token = newToken(type_var_start, code[cursor:cursor+2], line)
		}
		stream.tokens = append(stream.tokens, token)
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
				stream.tokens = append(stream.tokens, newToken(type_operator, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_word.FindStringIndex(code[cursor:end]); sPos != nil {
				word = code[cursor : cursor+sPos[1]]
				moveCursor(cursor + sPos[1])
				if isWordOperator(word) {
					stream.tokens = append(stream.tokens, newToken(type_operator, word, line))
					continue
				}
				stream.tokens = append(stream.tokens, newToken(type_name, word, line))
			} else if sPos = reg_number.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(type_number, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_string.FindStringIndex(code[cursor:end]); sPos != nil {
				str := strings.Trim(code[cursor:cursor+sPos[1]], "\"'")
				stream.tokens = append(stream.tokens, newToken(type_string, str, line))
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_punctuation.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(type_punctuation, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos = reg_bracket.FindStringIndex(code[cursor:end]); sPos != nil {
				bracket = code[cursor+sPos[0] : cursor+sPos[1]]
				if reg_bracket_open.MatchString(bracket) {
					brackets = append(brackets, &Bracket{ch: bracket, Line: line})
				} else if reg_bracket_close.MatchString(bracket) {
					if len(brackets) == 0 {
						return nil, &UnexpectedToken{Line: line, token: bracket}
					}
					switch {
					case brackets[len(brackets)-1].ch == "(" && bracket != ")":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					case brackets[len(brackets)-1].ch == "[" && bracket != "]":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					case brackets[len(brackets)-1].ch == "{" && bracket != "}":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					}
					brackets = brackets[:len(brackets)-1]
				}
				stream.tokens = append(stream.tokens, newToken(type_operator, bracket, line))
				moveCursor(cursor + sPos[1])
			} else {
				return nil, &UnexpectedToken{Line: line, token: code[cursor:end]}
			}
		}
		if len(brackets) > 0 {
			return nil, &UnClosedToken{Line: brackets[0].Line, token: brackets[0].ch}
		}
		moveCursor(end)
		if reg == reg_block {
			token = newToken(type_command_end, code[cursor:cursor+length], line)
		} else {
			token = newToken(type_var_end, code[cursor:cursor+length], line)
		}
		stream.tokens = append(stream.tokens, token)
		moveCursor(cursor + length)

		posIndex++
	}

	if cursor < codeLen {
		stream.tokens = append(stream.tokens, newToken(type_text, code[cursor:codeLen], line))
		moveCursor(codeLen)
	}

	stream.tokens = append(stream.tokens, newToken(type_eof, "", line))

	return stream, nil
}

func newToken(typ int, value string, line int) *token {
	return &token{typ: typ, value: value, line: line}
}

type Bracket struct {
	ch   string
	Line int
}

func (b *Bracket) String() string {
	return fmt.Sprintf("%s at line %d", b.ch, b.Line)
}

type TokenStream struct {
	Source  *sourceCode
	tokens  []*token
	current int
}

func (ts *TokenStream) Size() int {
	return len(ts.tokens)
}

func (ts *TokenStream) String() string {
	sb := &strings.Builder{}
	for _, t := range ts.tokens {
		sb.WriteString(t.string())
	}

	return sb.String()
}

func (ts *TokenStream) Current() (*token, error) {
	if ts.current >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.current], nil
}

func (ts *TokenStream) HasNext() bool {
	size := len(ts.tokens)

	return ts.current < size-1
}

func (ts *TokenStream) Next() (*token, error) {
	ts.current++
	if ts.current > len(ts.tokens)-1 {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.current], nil
}

func (ts *TokenStream) Skip(n int) (*token, error) {
	ts.current += n
	if ts.current >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.current], nil
}

func (ts *TokenStream) Peek(n int) (*token, error) {
	if ts.current+n >= len(ts.tokens)-1 {
		return nil, &UnexpectedEndOfFile{}
	}

	return ts.tokens[ts.current+n], nil
}

func isWordOperator(word string) bool {
	for _, v := range word_operators {
		if v == word {
			return true
		}
	}

	return false
}
