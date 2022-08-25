package template

import "fmt"

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
	return fmt.Sprintf("Un closed token \"%s\" in line %d", e.token, e.Line)
}

type UnexpectedToken struct {
	Line  int
	token string
}

func (e *UnexpectedToken) Error() string {
	return fmt.Sprintf("Un expected token \"%s\" in line %d", e.token, e.Line)
}
