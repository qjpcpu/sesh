package util

import (
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "sssh"
    "time"
)

type s3hrc struct {
    User    string
    Keyfile string
}

func Gets3hrc() (conf map[string]string, err error) {
    conf = make(map[string]string)
    fn := os.Getenv("HOME") + "/.seshrc"
    if _, err = os.Stat(fn); os.IsNotExist(err) {
        return conf, err
    }
    if buf, err := ioutil.ReadFile(fn); err != nil {
        return conf, err
    } else {
        rc := &s3hrc{}
        err = json.Unmarshal(buf, rc)
        if err != nil {
            return conf, err
        }
        conf["user"] = rc.User
        conf["keyfile"] = rc.Keyfile
        return conf, err
    }
}
func report(s3h *sssh.Sssh, host string, data interface{}) {
    if v, ok := data.(string); ok && v == "BEGIN" {
        s3h.Output.Write([]byte(fmt.Sprintf("\033[33m========== %s ==========\033[0m\n", host)))
    }
}
func SerialRun(config map[string]interface{}, host_arr []string) error {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    printer, _ := config["Output"].(io.Writer)

    s3h := &sssh.Sssh{
        User:         user,
        Password:     pwd,
        Keyfile:      keyfile,
        Cmd:          cmd,
        Output:       printer,
        StateChanged: report,
    }
    size := len(host_arr)
    queue := make(chan map[string]string, size)
    for _, h := range host_arr {
        s3h.Work(h, queue)
    }
    return nil
}
func ParallelRun(config map[string]interface{}, host_arr []string, tmpdir string) error {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    printer, _ := config["Output"].(io.Writer)

    size := len(host_arr)
    queue := make(chan map[string]string, size)

    dir := fmt.Sprintf("%s/.s3h.%d", tmpdir, time.Now().Second())
    if err := os.Mkdir(dir, os.ModeDir|os.ModePerm); err != nil {
        return err
    }
    var tmpfiles []*os.File
    for _, h := range host_arr {
        file, _ := os.Create(fmt.Sprintf("%s/%s", dir, h))
        tmpfiles = append(tmpfiles, file)
        s3h := &sssh.Sssh{
            User:         user,
            Password:     pwd,
            Keyfile:      keyfile,
            Cmd:          cmd,
            Output:       file,
            StateChanged: report,
        }
        go s3h.Work(h, queue)
    }

Loop:
    for {
        select {
        case msg := <-queue:
            if msg["BODY"] == "END" {
                size -= 1
            }
            if size == 0 {
                break Loop
            }
        }
    }
    // close tmp files
    for _, f := range tmpfiles {
        f.Close()
    }
    for _, h := range host_arr {
        fn := fmt.Sprintf("%s/%s", dir, h)
        src, _ := os.Open(fn)
        io.Copy(printer, src)
        src.Close()
        os.Remove(fn)
    }
    os.Remove(dir)
    return nil
}
