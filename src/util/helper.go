package util

import (
    "bytes"
    "strings"
)

func format_cmd(cmd string) string {
    if strings.HasPrefix(cmd, "#!") {
        buf := bytes.NewBufferString(cmd)
        if exe, err := buf.ReadString('\n'); err == nil {
            return "(cat<<\\EOF\n" + cmd + "\nEOF\n)|" + exe[2:]
        }
    }
    return cmd
}
