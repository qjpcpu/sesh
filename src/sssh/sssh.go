package sssh

import (
	"bufio"
	"fmt"
	. "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"io/ioutil"
	"job"
	"os"
	"os/exec"
	"os/signal"
	"strings"
    "net"
)

type Sssh struct {
	User     string
	Password string
	Keyfile  string
	Output   io.Writer
	Errout   io.Writer
	Cmd      string
	Host     string
	Timeout  int
	*job.Member
}

func NewS3h(host, user, password, keyfile, cmd string, output, err_out io.Writer, mgr *job.Member) (s3h *Sssh) {
	m, _ := mgr.NewMember(host)
	s3h = &Sssh{
		User:     user,
		Password: password,
		Keyfile:  keyfile,
		Output:   output,
		Errout:   err_out,
		Cmd:      cmd,
		Host:     host,
		Timeout:  5,
	}
	s3h.Member = m
	return
}
func getkey(file string) (key Signer, err error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ParsePrivateKey(buf)
	if err != nil {
		return
	}
	return

}
func (s3h *Sssh) Work() {
	if s3h.Member != nil {
		s3h.Send(job.MASTER_ID, map[string]interface{}{"FROM": s3h.Host, "BODY": "BEGIN", "TAG": s3h})
		defer func() {
			s3h.Send(job.MASTER_ID, map[string]interface{}{"FROM": s3h.Host, "BODY": "END"})
		}()
		// Wait for master's reply
		data, _ := s3h.Receive(-1)
		info, _ := data.(map[string]interface{})
		if info["FROM"].(string) != job.MASTER_ID || info["BODY"].(string) != "CONTINUE" {
			return
		}
	}
    ssh_agent := func() AuthMethod {
        if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
            return PublicKeysCallback(agent.NewClient(sshAgent).Signers)
        }
        return nil
    }
	auths := []AuthMethod{
		Password(s3h.Password),
        ssh_agent(),
	}
	if s3h.Keyfile != "" {
		if key, err := getkey(s3h.Keyfile); err == nil {
			auths = append(auths, PublicKeys(key))
		}
	}
	config := &ClientConfig{
		User:    s3h.User,
		Auth:    auths,
	}
	conn, err := Dial("tcp", s3h.Host+":22", config)
	if err != nil {
		if s3h.Password != "" && strings.Contains(err.Error(), "unable to authenticate, attempted methods [none publickey]") {
			config = &ClientConfig{
				User:    s3h.User,
				Auth:    []AuthMethod{Password(s3h.Password)},
			}
			conn, err = Dial("tcp", s3h.Host+":22", config)
			if err != nil {
				fmt.Fprintln(s3h.Errout, "unable to connect: ", err.Error())
				return
			}
		} else {
			fmt.Fprintln(s3h.Errout, "unable to connect: ", err.Error())
			return
		}
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintln(s3h.Errout, "Failed to create session: "+err.Error())
		return
	}
	defer session.Close()

	session.Stdout = s3h.Output
	session.Stderr = s3h.Output
	session.Run(s3h.Cmd)
}

func (s3h *Sssh) Login() {
    ssh_agent := func() AuthMethod {
        if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
            return PublicKeysCallback(agent.NewClient(sshAgent).Signers)
        }
        return nil
    }
	auths := []AuthMethod{
		Password(s3h.Password),
        ssh_agent(),
	}

	if s3h.Keyfile != "" {
		if key, err := getkey(s3h.Keyfile); err == nil {
			auths = append(auths, PublicKeys(key))
		}
	}
	config := &ClientConfig{
		User:    s3h.User,
		Auth:    auths,
	}
	conn, err := Dial("tcp", s3h.Host+":22", config)
	if err != nil {
		fmt.Fprintln(s3h.Errout, "unable to connect: ", err.Error())
		return
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintln(s3h.Errout, "Failed to create session: "+err.Error())
		return
	}
	defer session.Close()
	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, _ := session.StdinPipe()

	// Set up terminal modes
	modes := TerminalModes{
		ECHO:          0,     // disable echoing
		TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 200, modes); err != nil {
		fmt.Fprintln(s3h.Errout, "request for pseudo terminal failed: %s", err)
		return
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		fmt.Fprintln(s3h.Errout, "failed to start shell: %s", err)
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	qc := make(chan string)
	go func() {
	Loop:
		for {
			select {
			case <-c:
				session.Signal(SIGINT)
				fmt.Println("")
			case <-qc:
				signal.Stop(c)
				break Loop
			}
		}
	}()
	// Accepting commands
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprint(in, scanner.Text()+"\n")
	}
	qc <- "Quit signal monitor"
}
func (s3h *Sssh) SysLogin() {
	str := "ssh " + s3h.User + "@" + s3h.Host
	if s3h.Keyfile != "" {
		str = str + " -i " + s3h.Keyfile
	}
	parts := strings.Fields(str)
	head := parts[0]
	parts = parts[1:len(parts)]
	cmd := exec.Command(head, parts...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(s3h.Errout, err.Error())
		return
	}
}
