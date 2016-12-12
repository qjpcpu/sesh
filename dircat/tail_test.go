package dircat

import (
    "testing"
)

func TestTailFile(t *testing.T) {
    text, err := TailFile("./tail_test.go", 1)
    if err != nil {
        t.Error("tail file fail")
    }
    if text != "}" {
        t.Error("tail file content error")
    }
}
