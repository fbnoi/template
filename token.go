package template

const (
	type_text = iota
	type_command_start
	type_var_start
	type_command_end
	type_var_end
	type_name
	type_number
	type_bool
	type_string
	type_operator
	type_punctuation
)

type token struct {
	value string
	typ   int
	line  int
}

func (t *token) string() string {
	if t.typ == type_string {
		return "\"" + t.value + "\""
	}

	return t.value
}
