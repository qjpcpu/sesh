package util

import (
    "code.google.com/p/go.crypto/ssh/terminal"
    "cowsay"
    "dircat"
    "fmt"
    "github.com/cheggaaa/pb"
    cfg "goconf.googlecode.com/hg"
    "io"
    "io/ioutil"
    "job"
    "os"
    "os/signal"
    "path/filepath"
    "sssh"
    "time"
)

func Gets3hrc() (conf map[string]map[string]string, err error) {
    conf = make(map[string]map[string]string)
    fn := os.Getenv("HOME") + "/.seshrc"
    if _, err = os.Stat(fn); os.IsNotExist(err) {
        return conf, err
    }
    if c, err := cfg.ReadConfigFile(fn); err != nil {
        return conf, err
    } else {
        sections := c.GetSections()
        for _, sec := range sections {
            conf[sec] = make(map[string]string)
            if user, err := c.GetString(sec, "user"); err == nil {
                conf[sec]["user"] = user
            }
            if keyfile, err := c.GetString(sec, "keyfile"); err == nil {
                conf[sec]["keyfile"] = keyfile
            }
            if password, err := c.GetString(sec, "password"); err == nil {
                conf[sec]["password"] = password
            }
        }
        return conf, err
    }
}

func GirlSay(content ...interface{}) string {
    return cowsay.Format(fmt.Sprint(content))
}

// Hook for per task state changed
func report(output io.Writer, prefix, host string, color bool) {
    if color {
        output.Write([]byte(fmt.Sprintf("\033[33m%s========== %s ==========\033[0m\n", prefix, host)))
    } else {

        output.Write([]byte(fmt.Sprintf("%s========== %s ==========\n", prefix, host)))
    }
}
func SerialRun(config map[string]interface{}, raw_host_arr []string, start, end int) error {
    host_arr := raw_host_arr[start:end]
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    args, _ := config["Args"].(string)
    // Format command
    cmd = format_cmd(cmd, args)
    printer, _ := config["Output"].(io.Writer)

    mgr, _ := job.NewManager()

    //Setup progress bar if the output is not os.Stdout
    var bar *pb.ProgressBar
    if printer != os.Stdout {
        bar = pb.StartNew(len(host_arr))
    }
    for index, h := range host_arr {
        s3h := sssh.NewS3h(h, user, pwd, keyfile, cmd, printer, mgr)
        go func() {
            if _, err := mgr.Receive(-1); err == nil {
                report(s3h.Output, fmt.Sprintf("%d/%d ", index+1+start, len(raw_host_arr)), s3h.Host, os.Stdout == printer)
                mgr.Send(s3h.Host, map[string]interface{}{"FROM": "MASTER", "BODY": "CONTINUE"})
            } else {
                mgr.Send(s3h.Host, map[string]interface{}{"FROM": "MASTER", "BODY": "STOP"})
            }
        }()
        if printer != os.Stdout {
            bar.Increment()
        }
        s3h.Work()
    }
    if printer != os.Stdout {
        bar.FinishPrint("")
    }
    return nil
}
func ParallelRun(config map[string]interface{}, raw_host_arr []string, start, end int, tmpdir string) error {
    host_arr := raw_host_arr[start:end]
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    cmd, _ := config["Cmd"].(string)
    args, _ := config["Args"].(string)
    cmd = format_cmd(cmd, args)
    printer, _ := config["Output"].(io.Writer)

    // Create master, the master is used to manage go routines
    mgr, _ := job.NewManager()
    // Setup tmp directory for tmp files
    dir := fmt.Sprintf("%s/.s3h.%d", tmpdir, time.Now().Nanosecond())
    if err := os.Mkdir(dir, os.ModeDir|os.ModePerm); err != nil {
        return err
    }

    // Print cowsay wait
    //fmt.Println(girlSay("  Please wait me for a moment, Baby!  "))
    // Listen interrupt and kill signal, clear tmp files before exit.
    intqueue := make(chan os.Signal, 1)
    signal.Notify(intqueue, os.Interrupt, os.Kill)
    // If got interrupt or kill signal, delete tmp directory first, then exit with 1
    go func() {
        <-intqueue
        os.RemoveAll(dir)
        os.Exit(1)
    }()
    // If the complete all the tasks normlly, stop listenning signals and remove tmp directory
    defer func() {
        signal.Stop(intqueue)
        os.RemoveAll(dir)
    }()

    // Create tmp file for every host, then executes.
    var tmpfiles []*os.File
    for _, h := range host_arr {
        file, _ := os.Create(fmt.Sprintf("%s/%s", dir, h))
        tmpfiles = append(tmpfiles, file)
        s3h := sssh.NewS3h(h, user, pwd, keyfile, cmd, file, mgr)
        go s3h.Work()
    }

    // show realtime view for each host
    var dc *dircat.DirCat
    if terminal.IsTerminal(0) && printer == os.Stdout {
        wlist := []string{}
        for _, h := range host_arr {
            wlist = append(wlist, fmt.Sprintf("%s/%s", dir, h))
        }
        dc, _ = dircat.Init(wlist...)
        go dc.Start()
    }
    // When a host is ready and request for continue, the master would echo CONTINUE for response to allow host to run
    size := len(host_arr)
    for {
        data, _ := mgr.Receive(-1)
        info, _ := data.(map[string]interface{})
        if info["BODY"].(string) == "BEGIN" {
            report(info["TAG"].(*sssh.Sssh).Output, "", info["TAG"].(*sssh.Sssh).Host, printer == os.Stdout)
            mgr.Send(info["FROM"].(string), map[string]interface{}{"FROM": "MASTER", "BODY": "CONTINUE"})
        } else if info["BODY"].(string) == "END" {
            // If master gets every hosts' END message, then it stop waiting.
            size -= 1
            if size == 0 {
                break
            }
        }
    }
    if terminal.IsTerminal(0) && printer == os.Stdout {
        dc.Stop()
    }
    // close tmp files
    for _, f := range tmpfiles {
        f.Close()
    }
    // Merge all the hosts' output to the output file
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
    s3h.SysLogin()
}
func ScpRun(config map[string]interface{}, host_arr []string) error {
    user, _ := config["User"].(string)
    pwd, _ := config["Password"].(string)
    keyfile, _ := config["Keyfile"].(string)
    src, _ := config["Source"].(string)
    dest, _ := config["Destdir"].(string)

    animation := make(chan int)
    animation_on := true
    go func() {
        fmt.Print("\033[?25l")
        b := []string{"-", "\\", "|", "/", "-", "|", "/"}
        i := 0
        for {
            select {
            case <-animation:
                fmt.Print("\033[?25h")
                break
            case <-time.After(100 * time.Millisecond):
                i += 1
                if animation_on {
                    fmt.Printf("%v\r", b[i%7])
                }
            }
        }
    }()

    perm := "0660"
    if fi, err := os.Stat(src); err != nil {
        return err
    } else {
        perm = fmt.Sprintf("%#o", fi.Mode())
    }
    data, err := ioutil.ReadFile(src)
    if err != nil {
        return err
    }
    dest = dest + "/" + filepath.Base(src)
    // Create master, the master is used to manage go routines
    mgr, _ := job.NewManager()
    for _, h := range host_arr {
        scp := sssh.NewScp(h, user, pwd, keyfile, dest, perm, data, mgr)
        go scp.Work()
    }

    // When a host is ready and request for continue, the master would echo CONTINUE for response to allow host to run
    size := len(host_arr)
    for {
        data, _ := mgr.Receive(-1)
        info, _ := data.(map[string]interface{})
        if info["BODY"].(string) == "BEGIN" {
            mgr.Send(info["FROM"].(string), map[string]interface{}{"FROM": "MASTER", "BODY": "CONTINUE"})
        } else if info["BODY"].(string) == "END" {
            // If master gets every hosts' END message, then it stop waiting.
            if animation_on {
                animation_on = false
                animation <- 0
            }
            fmt.Print(info["RES"].(string))
            size -= 1
            if size == 0 {
                break
            }
        }
    }
    return nil
}
