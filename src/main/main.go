package main

import (
    "bufio"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "strings"
    "templ"
    "util"
)

type cfilesFlag []string

func (c *cfilesFlag) String() string {
    return fmt.Sprint(*c)
}
func (c *cfilesFlag) Set(value string) error {
    for _, v := range *c {
        if v == value {
            return nil
        }
    }
    *c = append(*c, value)
    return nil
}
func showHelp() {
    fmt.Println("\033[33mUsage: sesh -f HOST_FILE -u USER -p PASSWORD COMMAND\033[0m")
    fmt.Println("\033[33mSimple usage(with config file ~/.seshrc): sesh -f HOST_FILE COMMAND\033[0m")
    flag.PrintDefaults()
}
func parseData(data string) map[string]interface{} {
    kv := make(map[string]interface{})
    data = strings.Replace(data, " ", "", -1)
    data = strings.TrimSuffix(data, ",")
    if data == "" {
        return kv
    }
    arr := strings.Split(data, ",")
    for _, b := range arr {
        tmp := strings.Split(b, "=")
        kv[tmp[0]] = tmp[1]
    }
    return kv
}
func main() {
    hostfile := flag.String("f", "", "HOST_FILE, every host per line.")
    hostlist := flag.String("h", "", "HOSTS, hosts seperated by comma.")
    user := flag.String("u", "", "USER, user name.")
    password := flag.String("p", "", "PASSWORD, password.")
    keyfile := flag.String("k", "", "KEY_FILE, rsa file.")
    outfile := flag.String("o", "", "OUTFILE, Save output to file.")
    var cmd_file_list cfilesFlag
    flag.Var(&cmd_file_list, "c", "CMD_FILE, Command file.")
    tmpdir := flag.String("t", ".", "TMP_DIRECTORY, Specify tmp directory.")
    parallel := flag.Bool("parallel", false, "Parallel execution.")
    pause := flag.Bool("check", false, "Pause after first host done.")
    help := flag.Bool("help", false, "See help.")
    data := flag.String("d", "", "the name would be replace by v in command or command file. The name format in command should be {{ .name }}")
    flag.Parse()

    //show help
    if *help {
        showHelp()
        return
    }

    // get hosts
    var host_arr []string
    if *hostfile == "" && *hostlist == "" {
        fmt.Println("\033[31mPlese specify hosts with -f or -h!\033[0m")
        return
    }
    if *hostfile != "" {
        if buf, err := ioutil.ReadFile(*hostfile); err != nil {
            fmt.Println("\033[31mFailed to read host from file!\033[0m")
            return
        } else {
            hoststr := string(buf)
            hoststr = strings.Replace(hoststr, " ", "", -1)
            hoststr = strings.TrimSuffix(hoststr, "\n")
            host_arr = strings.Split(hoststr, "\n")
        }
    } else {
        hoststr := strings.TrimSuffix(*hostlist, ",")
        host_arr = strings.Split(hoststr, ",")
    }

    rc, err := util.Gets3hrc()
    if *user == "" {
        if err == nil {
            *user = rc["user"]
        }
        if *user == "" {
            *user = os.Getenv("USER")
        }
    }
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
    cmd := ""
    if len(cmd_file_list) > 0 {
        for _, cf := range cmd_file_list {
            if _, err := os.Stat(cf); os.IsNotExist(err) {
                fmt.Println("\033[31mCommand file " + cf + " not found!\033[0m")
                return
            }
        }
        if o, err := templ.ParseFromFiles(cmd_file_list, parseData(*data)); err != nil {
            fmt.Println("\033[31mParse command file failed!\033[0m")
            return
        } else {
            cmd = o
        }
    } else {
        for _, v := range flag.Args() {
            cmd = cmd + v + " "
        }
        if o, err := templ.ParseFromString(cmd, parseData(*data)); err != nil {
            fmt.Println("\033[31mParse command file failed!\033[0m")
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
        "Output":   printer,
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
        util.ParallelRun(config, host_arr[host_offset:len(host_arr)], *tmpdir)
    } else {
        util.SerialRun(config, host_arr[host_offset:len(host_arr)])
    }

    fmt.Println("\033[32mFinished!\033[0m")
}
