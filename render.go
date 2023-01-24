package template

import "io"

func Render(path string, writer io.Writer, ps Params) (err error) {
	doc, err := buildFileTemplate(path)
	if err != nil {
		return
	}
	body, err := doc.execute(ps)
	if err != nil {
		return
	}

	_, err = writer.Write([]byte(body))

	return
}

func RenderView(tpl string, writer io.Writer, ps Params) (err error) {
	doc, err := buildTemplate(tpl)
	if err != nil {
		return
	}
	body, err := doc.execute(ps)
	if err != nil {
		return
	}

	_, err = writer.Write([]byte(body))

	return
}
