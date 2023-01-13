package template

import "fmt"

func newUnexpectedToken(tok *token) error {
	return &UnexpectedToken{Line: tok.line, token: tok.value}
}

type UnexpectedEndOfFile struct {
}

func (e UnexpectedEndOfFile) Error() string {
	return "Unexpected end of file."
}

type UnClosedToken struct {
	Line  int
	token string
}

func (e *UnClosedToken) Error() string {
	return fmt.Sprintf("Unclosed token \"%s\" in line %d", e.token, e.Line)
}

type UnexpectedToken struct {
	Line  int
	token string
}

func (e *UnexpectedToken) Error() string {
	return fmt.Sprintf("Unexpected token \"%s\" in line %d", e.token, e.Line)
}
