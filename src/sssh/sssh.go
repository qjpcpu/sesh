package sssh

import (
    . "code.google.com/p/go.crypto/ssh"
    "io"
    "io/ioutil"
    "job"
    "log"
)

type Sssh struct {
    User     string
    Password string
    Keyfile  string
    Output   io.Writer
    Cmd      string
    Host     string
    *job.Member
}

func NewS3h(host, user, password, keyfile, cmd string, output io.Writer, mgr *job.Member) (s3h *Sssh) {
    m, _ := mgr.NewMember(host)
    s3h = &Sssh{
        user,
        password,
        keyfile,
        output,
        cmd,
        host,
        m,
    }
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
        s3h.Send("MASTER", map[string]interface{}{"FROM": s3h.Host, "BODY": "BEGIN", "TAG": s3h})
        // Wait for master's reply
        data, _ := s3h.Receive(-1)
        info, _ := data.(map[string]interface{})
        if info["FROM"].(string) != "MASTER" && info["BODY"].(string) != "CONTINUE" {
            return
        }
    }
    auths := []AuthMethod{
        Password(s3h.Password),
    }
    if s3h.Keyfile != "" {
        if key, err := getkey(s3h.Keyfile); err == nil {
            auths = append(auths, PublicKeys(key))
        }
    }
    config := &ClientConfig{
        User: s3h.User,
        Auth: auths,
    }
    conn, err := Dial("tcp", s3h.Host+":22", config)
    if err != nil {
        log.Fatalf("unable to connect: %s", err.Error())
    }
    defer conn.Close()
    session, err := conn.NewSession()
    if err != nil {
        panic("Failed to create session: " + err.Error())
    }
    defer session.Close()

    session.Stdout = s3h.Output
    session.Stderr = s3h.Output
    session.Run(s3h.Cmd)

    if s3h.Member != nil {
        s3h.Send("MASTER", map[string]interface{}{"FROM": s3h.Host, "BODY": "END"})
    }
}
