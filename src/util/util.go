package util

import (
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "job"
    "os"
    "os/signal"
    "sssh"
    "time"
)

// Get configurations form $HOME/.seshrc
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

// Hook for per task state changed
func report(output io.Writer, host string) {
    output.Write([]byte(fmt.Sprintf("\033[33m========== %s ==========\033[0m\n", host)))
}
func SerialRun(config map[string]interface{}, host_arr []string) error {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    printer, _ := config["Output"].(io.Writer)

    mgr, _ := job.NewManager()

    for _, h := range host_arr {
        s3h := sssh.NewS3h(h, user, pwd, keyfile, cmd, printer, mgr)
        go func() {
            if _, err := mgr.Receive(-1); err == nil {
                report(s3h.Output, s3h.Host)
                mgr.Send(s3h.Host, map[string]interface{}{"FROM": "MASTER", "BODY": "CONTINUE"})
            } else {
                mgr.Send(s3h.Host, map[string]interface{}{"FROM": "MASTER", "BODY": "STOP"})
            }
        }()
        s3h.Work()
    }
    return nil
}
func ParallelRun(config map[string]interface{}, host_arr []string, tmpdir string) error {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    printer, _ := config["Output"].(io.Writer)

    // Create master
    mgr, _ := job.NewManager()
    // Setup tmp directory for tmp files
    dir := fmt.Sprintf("%s/.s3h.%d", tmpdir, time.Now().Nanosecond())
    if err := os.Mkdir(dir, os.ModeDir|os.ModePerm); err != nil {
        return err
    }

    // Listen interrupt and kill signal, clear tmp files before exit.
    intqueue := make(chan os.Signal, 1)
    signal.Notify(intqueue, os.Interrupt, os.Kill)
    go func() {
        <-intqueue
        os.RemoveAll(dir)
        os.Exit(1)
    }()
    defer func() {
        signal.Stop(intqueue)
        os.RemoveAll(dir)
    }()

    var tmpfiles []*os.File
    for _, h := range host_arr {
        file, _ := os.Create(fmt.Sprintf("%s/%s", dir, h))
        tmpfiles = append(tmpfiles, file)
        s3h := sssh.NewS3h(h, user, pwd, keyfile, cmd, file, mgr)
        go s3h.Work()
    }

    size := len(host_arr)
    for {
        data, _ := mgr.Receive(-1)
        info, _ := data.(map[string]interface{})
        if info["BODY"].(string) == "BEGIN" {
            report(info["TAG"].(*sssh.Sssh).Output, info["TAG"].(*sssh.Sssh).Host)
            mgr.Send(info["FROM"].(string), map[string]interface{}{"FROM": "MASTER", "BODY": "CONTINUE"})
        } else if info["BODY"].(string) == "END" {
            size -= 1
            if size == 0 {
                break
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
    return nil
}
func Interact(config map[string]interface{}, host string) {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    printer, _ := config["Output"].(io.Writer)

    mgr, _ := job.NewManager()
    s3h := sssh.NewS3h(host, user, pwd, keyfile, cmd, printer, mgr)
    s3h.Login()
}
