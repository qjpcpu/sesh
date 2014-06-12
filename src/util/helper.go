package util

import (
    "bytes"
    "fmt"
    "strings"
    "time"
)

func format_cmd(cmd, args string) string {
    if strings.HasPrefix(cmd, "#!") {
        buf := bytes.NewBufferString(cmd)
        if exe, err := buf.ReadString('\n'); err == nil {
            exe = strings.TrimRight(exe[2:], "\n")
            if args != "" {
                tmp := fmt.Sprintf("/tmp/sesh-%v", time.Now().Nanosecond())
                return "(cat >" + tmp + " <<\\EOF\n" + cmd + "\nEOF\n) && " + exe + " " + tmp + " " + args + "; rm -f " + tmp
            } else {
                return "(cat <<\\EOF\n" + cmd + "\nEOF\n) |" + exe
            }
        }
    }
    return cmd
}
