package template

import (
	"os"
	"path/filepath"

	v2 "gopkg.in/yaml.v2"
)

var pwd string

type Config struct {
	TplDir  string `yaml:"TplDir"`
	ExtName string `yaml:"ExtName"`
}

func InitConfig(path string) (err error) {
	pwd, err = os.Getwd()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(pwd, path))
	if err != nil {
		return
	}
	config = new(Config)
	err = v2.Unmarshal(data, config)

	return
}
