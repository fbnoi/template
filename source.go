package template

import (
	"crypto/sha1"
	"encoding/hex"
	"io/ioutil"
)

type Source struct {
	Identity string
	Code     string
}

type Line struct {
	Num       int
	Code      string
	Highlight bool
}

//Overview returns nearby code
func (s *Source) Overview(line int) (codes []*Line) {
	pos := reg_enter.FindAllStringIndex(s.Code, -1)
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
		codes = append(codes, &Line{Num: i, Code: s.Code[pos[i-1][1]:pos[i][0]], Highlight: i == line})
	}
	return
}

func NewSource(code string) *Source {
	return &Source{Code: code, Identity: abstract([]byte(code))}
}

func NewSourceFile(path string) *Source {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return &Source{Code: string(bs), Identity: path}
}

func abstract(content []byte) string {
	encryptor := sha1.New()
	encryptor.Write(content)
	return hex.EncodeToString(encryptor.Sum(nil))
}
