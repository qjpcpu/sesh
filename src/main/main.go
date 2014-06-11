package main

import (
    "bufio"
    "code.google.com/p/go.crypto/ssh/terminal"
    "fmt"
    "io/ioutil"
    flag "mflag"
    "os"
    "templ"
    "util"
)

func main() {
    hostfile := flag.String([]string{"f", "-host-file"}, "", "HOST_FILE, every host per line.")
    hostlist := flag.String([]string{"h", "-host-list"}, "", "HOSTS, hosts seperated by comma.")
    user := flag.String([]string{"u", "-user"}, "", "USER, user name.")
    password := flag.String([]string{"p", "-password"}, "", "PASSWORD, password.")
    keyfile := flag.String([]string{"k", "-key"}, "", "KEY_FILE, rsa file.")
    outfile := flag.String([]string{"o", "-output"}, "", "OUTFILE, Save output to file.")
    var cmd_file_list cfilesFlag
    flag.Var(&cmd_file_list, []string{"c", "-command-file"}, "CMD_FILE, Command file.")
    tmpdir := flag.String([]string{"t", "-tmp-directory"}, ".", "TMP_DIRECTORY, Specify tmp directory.")
    parallel := flag.Bool([]string{"r", "-rapid"}, false, "Parallel execution.")
    concurence := flag.Int([]string{"-parallel-degree"}, 0, "Parallel degree, default is the size of hosts.")
    pause := flag.Bool([]string{"-check"}, false, "Pause after first host done.")
    help := flag.Bool([]string{"-help"}, false, "See help.")
    data := flag.String([]string{"d", "-data"}, "", "the name would be replace according name=value pair in command or command file. The name format in command should be {{ .name }}")
    args := flag.String([]string{"-args"}, "", "args for script.")
    debug := flag.Bool([]string{"-debug"}, false, "Print the configurations, not perform tasks.")
    flag.Parse()

    //show help
    if *help {
        showHelp()
        return
    }

    // get hosts
    var host_arr []string
    if *hostfile != "" {
        if buf, err := ioutil.ReadFile(*hostfile); err != nil {
            fmt.Println("\033[31mFailed to read host from file!\033[0m")
            return
        } else {
            hoststr := string(buf)
            host_arr = parseHostsFromString(hoststr)
        }
    } else if *hostlist != "" {
        host_arr = parseHostsFromString(*hostlist)
    } else {
        if terminal.IsTerminal(0) {
            fmt.Println("\033[33mPlease input hosts, seperated by LINE SEPERATOR, press Ctrl+D to finish input:\033[0m")
        }
        buf, _ := ioutil.ReadAll(os.Stdin)
        host_arr = parseHostsFromString(string(buf))
    }

    // get user
    rc, err := util.Gets3hrc()
    if *user == "" {
        if err == nil {
            *user = rc["user"]
        }
        if *user == "" {
            *user = os.Getenv("USER")
        }
    }

    // get password or key file
    if *password == "" && *keyfile == "" {
        if err == nil {
            *keyfile = rc["keyfile"]
        }
        if *keyfile == "" {
            *keyfile = os.Getenv("HOME") + "/.ssh/id_rsa"
        }
        if _, err := os.Stat(*keyfile); os.IsNotExist(err) {
            fmt.Println("\033[31mKey file " + *keyfile + " not found!\033[0m")
            return
        }
    }

    //check command
    if len(flag.Args()) == 0 && len(cmd_file_list) == 0 {
        fmt.Println("\033[31mPlese specify command you want execute.\033[0m")
        return
    }
    // parse command template
    cmd := ""
    if len(cmd_file_list) > 0 {
        for _, cf := range cmd_file_list {
            if _, err := os.Stat(cf); os.IsNotExist(err) {
                fmt.Println("\033[31mCommand file " + cf + " not found!\033[0m")
                return
            }
        }
        if o, err := templ.ParseFromFiles(cmd_file_list, parseData(*data)); err != nil {
            fmt.Printf("\033[31mParse command file failed!\033[0m\n%v\n", err)
            return
        } else {
            cmd = o
        }
    } else {
        // join commands
        for _, v := range flag.Args() {
            cmd = cmd + v + " "
        }
        if o, err := templ.ParseFromString(cmd, parseData(*data)); err != nil {
            fmt.Printf("\033[31mParse command failed!\033[0m\n%v\n", err)
            return
        } else {
            cmd = o
        }
    }
    if _, err := os.Stat(*tmpdir); os.IsNotExist(err) && *parallel {
        fmt.Println("\033[31mTemporary directory " + *tmpdir + " is not exist!\033[0m")
        return
    }

    // Begin to run
    printer := os.Stdout
    if *outfile != "" {
        if output, err := os.Create(*outfile); err != nil {
            fmt.Println("\033[31mCan't create " + *outfile + "!\033[0m")
            return
        } else {
            printer = output
            defer printer.Close()
        }
    }
    config := map[string]interface{}{
        "User":     *user,
        "Password": *password,
        "Keyfile":  *keyfile,
        "Cmd":      cmd,
        "Args":     *args,
        "Output":   printer,
    }
    if *debug {
        printDebugInfo(config, host_arr)
        return
    }
    host_offset := 0
    if *pause {
        util.SerialRun(config, host_arr[0:1])
        fmt.Printf("The task on \033[33m%s\033[0m has done.\nPress any key to auto login \033[33m%s\033[0m to have a check...", host_arr[0], host_arr[0])
        reader := bufio.NewReader(os.Stdin)
        reader.ReadString('\n')
        util.Interact(config, host_arr[0])
        fmt.Printf("\n\033[32mCheck completed! Press any key to acomplish the left tasks.\033[0m")
        reader = bufio.NewReader(os.Stdin)
        reader.ReadString('\n')
        host_offset = 1
    }
    if *parallel {
        fmt.Println(util.GirlSay("  Please wait me for a moment, Baby!  "))
        end := len(host_arr)
        if *concurence < 1 || *concurence > (end-host_offset) {
            *concurence = end - host_offset
        }
        for {
            to := host_offset + *concurence
            if to > end {
                to = end
            }
            if host_offset >= to {
                break
            }
            util.ParallelRun(config, host_arr[host_offset:to], *tmpdir)
            host_offset += *concurence
        }
    } else {
        util.SerialRun(config, host_arr[host_offset:len(host_arr)])
    }

    fmt.Println("\033[32mFinished!\033[0m")
}
