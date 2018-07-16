package templ

import (
	"github.com/hoisie/mustache"
	"io/ioutil"
)

func ParseFromFiles(fn string, data map[string]interface{}) (string, error) {
	cmd := ""
	if buf, err := ioutil.ReadFile(fn); err != nil {
		return "", err
	} else {
		cmd = string(buf)
	}
	return ParseFromString(cmd, data)
}

func ParseFromString(cmd string, data map[string]interface{}) (string, error) {
	return mustache.Render(cmd, data), nil
}
