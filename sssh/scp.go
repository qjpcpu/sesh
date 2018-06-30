package sssh

import (
	"fmt"
	. "github.com/qjpcpu/sesh/golang.org/x/crypto/ssh"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh/agent"
	"github.com/qjpcpu/sesh/job"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type Scp struct {
	Host     string
	User     string
	Password string
	Keyfile  string
	Destfile string
	Perm     string
	Data     []byte
	Timeout  int
	*job.Member
}

func NewScp(host, user, password, keyfile, destfile, perm string, data []byte, mgr *job.Member) (scp *Scp) {
	m, _ := mgr.NewMember(host)
	scp = &Scp{
		host,
		user,
		password,
		keyfile,
		destfile,
		perm,
		data,
		5,
		m,
	}
	return
}
func (scp *Scp) Work() {
	res := map[string]interface{}{"FROM": scp.Host, "BODY": "END"}
	if scp.Member != nil {
		scp.Send(job.MASTER_ID, map[string]interface{}{"FROM": scp.Host, "BODY": "BEGIN", "TAG": scp})
		defer func() {
			scp.Send(job.MASTER_ID, res)
		}()
		// Wait for master's reply
		data, _ := scp.Receive(-1)
		info, _ := data.(map[string]interface{})
		if info["FROM"].(string) != job.MASTER_ID && info["BODY"].(string) != "CONTINUE" {
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
		Password(scp.Password),
	}
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		auths = append(auths, ssh_agent())
	}
	if scp.Keyfile != "" {
		if key, err := getkey(scp.Keyfile); err == nil {
			auths = append(auths, PublicKeys(key))
		}
	}
	config := &ClientConfig{
		User: scp.User,
		Auth: auths,
	}
	conn, err := Dial("tcp", scp.Host+":22", config)
	if err != nil {
		if scp.Password != "" && strings.Contains(err.Error(), "unable to authenticate, attempted methods [none publickey]") {
			config = &ClientConfig{
				User: scp.User,
				Auth: []AuthMethod{Password(scp.Password)},
			}
			conn, err = Dial("tcp", scp.Host+":22", config)
			if err != nil {
				res["RES"] = fmt.Sprintf("Unable to connect \033[31m%v\033[0m %v\n", scp.Host, err)
				return
			}
		} else {
			res["RES"] = fmt.Sprintf("Unable to connect \033[31m%v\033[0m %v\n", scp.Host, err)
			return
		}
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		res["RES"] = fmt.Sprintf("\033[31m%v\033[0m %v\n", scp.Host, err)
		return
	}
	defer session.Close()
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%v %v %v\n", scp.Perm, len(scp.Data), filepath.Base(scp.Destfile))
		w.Write(scp.Data)
		fmt.Fprint(w, "\x00")
	}()
	if err := session.Run("/usr/bin/scp -qrt " + filepath.Dir(scp.Destfile)); err != nil {
		res["RES"] = fmt.Sprintf("\033[31mFailed to copy to %v!\033[0m\n", scp.Host)
	} else {
		res["RES"] = fmt.Sprintf("\033[33m%v\033[0m OK!\n", scp.Host)
	}
}
