package sssh

import (
    "os"
    "testing"
)

func TestPasswordOk(t *testing.T) {
    s3h := &Sssh{
        User:     "qujp",
        Password: "Linux.2005522",
        Output:   os.Stdout,
        Cmd:      "echo login with Password",
    }
    s3h.Work("st01-sdcop-dev.st01")
}
func TestKeyOk(t *testing.T) {
    s3h := &Sssh{
        User:     "qujp",
        Password: "Linux.2005522",
        Keyfile:  "/home/work/.ssh/id_rsa",
        Output:   os.Stdout,
        Cmd:      "echo login with Keyfile",
    }
    s3h.Work("st01-sdcop-dev.st01")
}
