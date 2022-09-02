package template

import (
	"fmt"
	"regexp"
)

var (
	TAG_COMMENT         = [...]string{`{#`, `#}`}
	TAG_BLOCK           = [...]string{`{%`, `%}`}
	TAG_VARIABLE        = [...]string{`{{`, `}}`}
	TAG_ESCAPE_COMMENT  = [...]string{`@{#`, `#}`}
	TAG_ESCAPE_BLOCK    = [...]string{`@{%`, `%}`}
	TAG_ESCAPE_VARIABLE = [...]string{`@{{`, `}}`}
)

var (
	// }}
	reg_variable = regexp.MustCompile(fmt.Sprintf(`\s*%s`, TAG_VARIABLE[1]))
	// %}
	reg_block = regexp.MustCompile(fmt.Sprintf(`\s*%s`, TAG_BLOCK[1]))
	// #}
	reg_comment = regexp.MustCompile(fmt.Sprintf(`\s*%s`, TAG_COMMENT[1]))
	// {{ or {% or {#
	reg_token_start = regexp.MustCompile(fmt.Sprintf(`(@?%s|@?%s|@?%s)`, TAG_VARIABLE[0], TAG_BLOCK[0], TAG_COMMENT[0]))
	// \r\n \n
	reg_enter = regexp.MustCompile(`(\r\n|\n)`)
	// whitespace
	reg_whitespace = regexp.MustCompile(`^\s+`)
	// + - * / > < = and or
	reg_operator = regexp.MustCompile(`^[\+\-*\/><=:]{1,3}|^(and)|^(or)|^(in)`)
	// name
	reg_name = regexp.MustCompile(`^[a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*(\.[a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*)*`)
	// number
	reg_number = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?([Ee][\+\-][0-9]+)?`)
	// punctuation
	reg_punctuation   = regexp.MustCompile(`^[\(\)\[\]\{\}\?\:;,\|]`)
	reg_bracket_open  = regexp.MustCompile(`^[\{\[\(]$`)
	reg_bracket_close = regexp.MustCompile(`^[\}\]\)]$`)
	// string
	reg_string = regexp.MustCompile(`^"([^"\\\\]*(?:\\\\.[^"\\\\]*)*)"|^'([^\'\\\\]*(?:\\\\.[^\'\\\\]*)*)'`)
)

func Tokenize(source *Source) (*TokenStream, error) {
	var (
		code     = reg_enter.ReplaceAllString(source.Code, "\n")
		stream   = &TokenStream{Source: source, current: 0}
		poss     = reg_token_start.FindAllStringIndex(code, -1)
		cursor   = 0
		line     = 0
		posIndex = 0
		end      = len(code)
	)

	moveCursor := func(n int) {
		cursor = n
		line = len(reg_enter.FindAllString(code[:n], -1)) + 1
	}

	if len(poss) == 0 {
		stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:], line))
		cursor = len(code)
	}
	for posIndex < len(poss) {
		pos := poss[posIndex]
		if pos[0] < cursor {
			posIndex++
			continue
		} else if pos[0] > cursor {
			stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:pos[0]], line))
			moveCursor(pos[0])
		}
		var reg *regexp.Regexp
		switch code[pos[0]:pos[1]] {
		case TAG_ESCAPE_COMMENT[0]:
			moveCursor(pos[0] + 1)
			ends := reg_comment.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: TAG_ESCAPE_COMMENT[0]}
			}
			stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case TAG_ESCAPE_BLOCK[0]:
			moveCursor(pos[0] + 1)
			ends := reg_block.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: TAG_ESCAPE_BLOCK[0]}
			}
			stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case TAG_ESCAPE_VARIABLE[0]:
			moveCursor(pos[0] + 1)
			ends := reg_variable.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: TAG_ESCAPE_VARIABLE[0]}
			}
			stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case TAG_COMMENT[0]:
			ends := reg_comment.FindStringIndex(code[cursor:])
			if ends == nil {
				return nil, &UnClosedToken{Line: line, token: TAG_COMMENT[0]}
			}
			stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:cursor+ends[1]], line))
			moveCursor(cursor + ends[1])
		case TAG_BLOCK[0]:
			reg = reg_block

		case TAG_VARIABLE[0]:
			reg = reg_variable
		}
		var token *Token
		if reg == reg_block {
			token = newToken(TYPE_COMMAND_START, code[cursor:cursor+2], line)
		} else {
			token = newToken(TYPE_VAR_START, code[cursor:cursor+2], line)
		}
		stream.tokens = append(stream.tokens, token)
		moveCursor(cursor + 2)
		ends := reg.FindStringIndex(code[cursor:])
		if ends == nil {
			return nil, &UnClosedToken{Line: line, token: TAG_BLOCK[0]}
		}
		length := ends[1] - ends[0]
		end = cursor + ends[0]
		var brackets []*Bracket
		for cursor < end {
			if sPos := reg_whitespace.FindStringIndex(code[cursor:end]); sPos != nil {
				moveCursor(cursor + sPos[1])
				continue
			}
			if sPos := reg_operator.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(TYPE_OPERATOR, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos := reg_name.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(TYPE_NAME, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos := reg_number.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(TYPE_NUMBER, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos := reg_string.FindStringIndex(code[cursor:end]); sPos != nil {
				stream.tokens = append(stream.tokens, newToken(TYPE_STRING, code[cursor:cursor+sPos[1]], line))
				moveCursor(cursor + sPos[1])
			} else if sPos := reg_punctuation.FindStringIndex(code[cursor:end]); sPos != nil {
				bracket := code[cursor+sPos[0] : cursor+sPos[1]]
				if reg_bracket_open.MatchString(bracket) {
					brackets = append(brackets, &Bracket{ch: bracket, Line: line})
				} else if reg_bracket_close.MatchString(bracket) {
					if len(brackets) == 0 {
						return nil, &UnexpectedToken{Line: line, token: bracket}
					}
					opBracket := brackets[len(brackets)-1]
					switch {
					case opBracket.ch == "{" && bracket != "}":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					case opBracket.ch == "(" && bracket != ")":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					case opBracket.ch == "[" && bracket != "]":
						return nil, &UnexpectedToken{Line: line, token: bracket}
					}
					brackets = brackets[:len(brackets)-1]
				}
				stream.tokens = append(stream.tokens, newToken(TYPE_PUNCTUATION, bracket, line))
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
			token = newToken(TYPE_COMMAND_END, code[cursor:cursor+length], line)
		} else {
			token = newToken(TYPE_VAR_END, code[cursor:cursor+length], line)
		}
		stream.tokens = append(stream.tokens, token)
		moveCursor(cursor + length)

		posIndex++
	}

	if cursor < end {
		stream.tokens = append(stream.tokens, newToken(TYPE_TEXT, code[cursor:end], line))
		moveCursor(end)
	}

	stream.tokens = append(stream.tokens, newToken(TYPE_EOF, "", line))

	return stream, nil
}

func newToken(typ int, value string, line int) *Token {
	return &Token{typ: typ, value: value, line: line}
}

type Bracket struct {
	ch   string
	Line int
}

func (b *Bracket) String() string {
	return fmt.Sprintf("%s at line %d", b.ch, b.Line)
}

type TokenStream struct {
	Source  *Source
	tokens  []*Token
	current int
}

func (ts *TokenStream) String() string {
	return ts.Source.Code
}

func (ts *TokenStream) Current() (*Token, error) {
	if ts.current >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}
	return ts.tokens[ts.current-1], nil
}

func (ts *TokenStream) Next() (*Token, error) {
	ts.current++
	if ts.current >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}
	return ts.tokens[ts.current-1], nil
}

func (ts *TokenStream) Peek(n int) (*Token, error) {
	if ts.current+n >= len(ts.tokens) {
		return nil, &UnexpectedEndOfFile{}
	}
	return ts.tokens[ts.current+n], nil
}

func (ts *TokenStream) IsEOF() bool {
	return TYPE_EOF == ts.tokens[ts.current].Type()
}

func (ts *TokenStream) SubStream(start, end int) TokenStream
