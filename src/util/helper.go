package util

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func format_cmd(cmd, args string) string {
	// dump command to file then execute the script file if args is not empty
	// the args should look like 'bash {} arg1 arg2' or '{} arg1 arg2', '{}' is script filename
	if args != "" {
		tmp := fmt.Sprintf("/tmp/sesh-%v", time.Now().Nanosecond())
		if strings.Contains(args, "{}") {
			args = strings.Replace(args, "{}", tmp, 1)
		} else {
			args = fmt.Sprintf("%s %s", tmp, args)
		}
		return fmt.Sprintf("(cat > %s  <<\\EOF\n%s\nEOF\n) && chmod +x %s && (%s) ; rm -f %s", tmp, cmd, tmp, args, tmp)
	} else {
		if strings.HasPrefix(cmd, "#!") {
			buf := bytes.NewBufferString(cmd)
			exe, _ := buf.ReadString('\n')
			exe = strings.TrimRight(exe[2:], "\n")
			return "(cat <<\\EOF\n" + cmd + "\nEOF\n) |" + exe
		} else {
			return cmd
		}
	}
}
