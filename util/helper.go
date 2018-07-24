package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

type BuildinCmd string

const (
	AuthCmdFile BuildinCmd = "@auth.cmd"
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

func (bc BuildinCmd) Get(pubkey_file string) (string, error) {
	d, err := ioutil.ReadFile(pubkey_file)
	if err != nil {
		return "", err
	}
	cmdstr := fmt.Sprintf(`
[ ! -e ~/.ssh ] && mkdir -p ~/.ssh && chmod 700 ~/.ssh
[ ! -e ~/.ssh/authorized_keys ] && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys
grep '%s' ~/.ssh/authorized_keys >/dev/null || echo '%s' >> ~/.ssh/authorized_keys
echo 'add public key to ~/.ssh/authorized_keys OK'
`, string(d), string(d))
	return cmdstr, nil
}
