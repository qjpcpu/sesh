package sssh

import (
    . "code.google.com/p/go.crypto/ssh"
    "fmt"
    "job"
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
        m,
    }
    return
}
func (scp *Scp) Work() {
    res := map[string]interface{}{"FROM": scp.Host, "BODY": "END"}
    if scp.Member != nil {
        scp.Send("MASTER", map[string]interface{}{"FROM": scp.Host, "BODY": "BEGIN", "TAG": scp})
        defer func() {
            scp.Send("MASTER", res)
        }()
        // Wait for master's reply
        data, _ := scp.Receive(-1)
        info, _ := data.(map[string]interface{})
        if info["FROM"].(string) != "MASTER" && info["BODY"].(string) != "CONTINUE" {
            return
        }
    }
    auths := []AuthMethod{
        Password(scp.Password),
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
