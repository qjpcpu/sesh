package dircat

import (
    "os"
    "strings"
)

const SizeOf100Lines = 120000

func TailBySize(filename string, unit int64) (string, error) {
    fi, err := os.Stat(filename)
    if err != nil {
        return "", err
    }
    size := fi.Size()
    file, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer file.Close()
    offset := size - unit
    if offset < 0 {
        offset = 0
    }
    file.Seek(offset, 0)
    data := make([]byte, unit)
    _, err = file.Read(data)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
func GetLastLines(text string, cnt int) string {
    text = strings.TrimRight(text, "\n")
    linecnt := strings.Count(text, "\n") + 1
    if linecnt > cnt {
        offset := 0
        for i := 0; i < linecnt-cnt; i++ {
            offset = strings.Index(text, "\n") + 1
            text = text[(offset):]
        }
    }
    return text
}
