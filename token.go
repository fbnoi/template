package template

const (
	TYPE_EOF = iota - 1
	TYPE_TEXT
	TYPE_COMMAND_START
	TYPE_VAR_START
	TYPE_COMMAND_END
	TYPE_VAR_END
	TYPE_NAME
	TYPE_NUMBER
	TYPE_STRING
	TYPE_OPERATOR
	TYPE_PUNCTUATION
)

type token struct {
	value string
	typ   int
	line  int
}

func (t *token) string() string {
	if t.typ == TYPE_STRING {
		return "\"" + t.value + "\""
	}

	return t.value
}
