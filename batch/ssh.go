package batch

import (
	"bufio"
	"fmt"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh/agent"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
)

type Task struct {
	User     string
	Password string
	Port     string
	Keyfile  string
	Output   io.Writer
	Errout   io.Writer
	Cmd      string
	Host     string
	Timeout  int
	*sync.WaitGroup
}

func NewTask(host, user, password, keyfile, cmd, port string, output, err_out io.Writer, wg *sync.WaitGroup) (task *Task) {
	wg.Add(1)
	if port == "" {
		port = "22"
	}
	task = &Task{
		User:     user,
		Password: password,
		Keyfile:  keyfile,
		Port:     port,
		Output:   output,
		Errout:   err_out,
		Cmd:      cmd,
		Host:     host,
		Timeout:  5,
	}
	task.WaitGroup = wg
	return
}

func getkey(file string) (key ssh.Signer, err error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	key, err = ssh.ParsePrivateKey(buf)
	if err != nil {
		return
	}
	return

}
func (task *Task) Work() {
	if task.WaitGroup != nil {
		defer task.Done()
	}
	ssh_agent := func() ssh.AuthMethod {
		if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
		}
		return nil
	}
	auths := []ssh.AuthMethod{
		ssh.Password(task.Password),
	}
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		if sagt := ssh_agent(); sagt != nil {
			auths = append(auths, ssh_agent())
		}
	}
	if task.Keyfile != "" {
		if key, err := getkey(task.Keyfile); err == nil {
			auths = append(auths, ssh.PublicKeys(key))
		}
	}
	config := &ssh.ClientConfig{
		User: task.User,
		Auth: auths,
	}
	conn, err := ssh.Dial("tcp", task.Host+":"+task.Port, config)
	if err != nil {
		if task.Password != "" && strings.Contains(err.Error(), "unable to authenticate, attempted methods [none publickey]") {
			config = &ssh.ClientConfig{
				User: task.User,
				Auth: []ssh.AuthMethod{ssh.Password(task.Password)},
			}
			conn, err = ssh.Dial("tcp", task.Host+":"+task.Port, config)
			if err != nil {
				fmt.Fprintln(task.Errout, "unable to connect: ", err.Error())
				return
			}
		} else {
			fmt.Fprintln(task.Errout, "unable to connect: ", err.Error())
			return
		}
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintln(task.Errout, "Failed to create session: "+err.Error())
		return
	}
	defer session.Close()

	session.Stdout = task.Output
	session.Stderr = task.Output
	session.Run(task.Cmd)
}

func (task *Task) Login() {
	ssh_agent := func() ssh.AuthMethod {
		if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
		}
		return nil
	}
	auths := []ssh.AuthMethod{
		ssh.Password(task.Password),
	}
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		if sagt := ssh_agent(); sagt != nil {
			auths = append(auths, ssh_agent())
		}
	}
	if task.Keyfile != "" {
		if key, err := getkey(task.Keyfile); err == nil {
			auths = append(auths, ssh.PublicKeys(key))
		}
	}
	config := &ssh.ClientConfig{
		User: task.User,
		Auth: auths,
	}
	conn, err := ssh.Dial("tcp", task.Host+":"+task.Port, config)
	if err != nil {
		fmt.Fprintln(task.Errout, "unable to connect: ", err.Error())
		return
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		fmt.Fprintln(task.Errout, "Failed to create session: "+err.Error())
		return
	}
	defer session.Close()
	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, _ := session.StdinPipe()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 200, modes); err != nil {
		fmt.Fprintln(task.Errout, "request for pseudo terminal failed: ", err)
		return
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		fmt.Fprintln(task.Errout, "failed to start shell: ", err)
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
				session.Signal(ssh.SIGINT)
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
func (task *Task) SysLogin() {
	if task.WaitGroup != nil {
		defer task.Done()
	}
	str := "ssh " + task.User + "@" + task.Host
	if task.Keyfile != "" {
		str = str + " -i " + task.Keyfile
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
		fmt.Fprintln(task.Errout, err.Error())
		return
	}
}
