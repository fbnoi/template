package template

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

var (
	config *Config
)

func Render(path string, writer io.Writer, ps Params) (err error) {
	path = resolvePath(path)
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

func WarmUp() (err error) {
	if config == nil || config.TplDir == "" || config.ExtName == "" {
		return errors.New("template: template dir or template extension name isn't configured")
	}
	suffix := "." + config.ExtName
	err = filepath.Walk(filepath.Join(pwd, config.TplDir), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), suffix) {
				if _, err = buildFileTemplate(path); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return
}

func resolvePath(p string) string {
	if p == "" {
		return p
	}
	if config == nil {
		p = filepath.Join(pwd, p)
	} else {
		p = filepath.Join(pwd, config.TplDir, p)
	}

	return path.Clean(p)
}
