package templ

import (
    "bytes"
    "io/ioutil"
    "text/template"
)

func ParseFromFiles(files []string, data map[string]interface{}) (string, error) {
    cmd := ""
    for _, fn := range files {
        if buf, err := ioutil.ReadFile(fn); err != nil {
            return "", err
        } else {
            cmd = cmd + string(buf)
        }
    }
    return ParseFromString(cmd, data)
}

func ParseFromString(cmd string, data map[string]interface{}) (string, error) {
    tmpl := template.Must(template.New("commands").Parse(cmd))
    var pcmd bytes.Buffer
    err := tmpl.Execute(&pcmd, data)
    return pcmd.String(), err
}
