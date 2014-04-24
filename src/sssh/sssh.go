package sssh

import (
    . "code.google.com/p/go.crypto/ssh"
    "io"
    "io/ioutil"
    "log"
)

type Sssh struct {
    User         string
    Password     string
    Keyfile      string
    Output       io.Writer
    Cmd          string
    StateChanged func(s3h *Sssh, host string, data interface{})
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
func (s3h *Sssh) Work(host string, queue chan map[string]string) {
    if s3h.StateChanged != nil {
        s3h.StateChanged(s3h, host, "BEGIN")
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
    conn, err := Dial("tcp", host+":22", config)
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

    if s3h.StateChanged != nil {
        s3h.StateChanged(s3h, host, "END")
    }
    queue <- map[string]string{"HOST": host, "BODY": "END"}
}
