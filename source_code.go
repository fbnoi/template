package template

import (
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
)

type sourceCode struct {
	identity string
	code     string
}

type textLine struct {
	num       int
	code      string
	highlight bool
}

//Overview returns nearby code
func (s *sourceCode) Overview(line int) (codes []*textLine) {
	pos := reg_enter.FindAllStringIndex(s.code, -1)
	len := len(pos)
	if line < 0 {
		return nil
	}
	if len < line || len == 0 {
		return nil
	}
	var (
		startLine, endLine int
	)
	endLine = line + 5
	if endLine > len {
		endLine = len
	}
	startLine = line - 5
	if startLine < 1 {
		startLine = 1
	}
	for i := startLine; i < endLine; i++ {
		codes = append(codes, &textLine{num: i, code: s.code[pos[i-1][1]:pos[i][0]], highlight: i == line})
	}

	return
}

func NewSourceCode(code string) *sourceCode {
	return &sourceCode{code: code, identity: abstract([]byte(code))}
}

func NewSourceCodeFile(path string) (*sourceCode, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &sourceCode{code: string(bs), identity: path}, nil
}

func abstract(content []byte) string {
	encryptor := sha1.New()
	encryptor.Write(content)

	return hex.EncodeToString(encryptor.Sum(nil))
}