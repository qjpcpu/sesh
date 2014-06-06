package main

import (
    "fmt"
    flag "mflag"
    "os"
    "strings"
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
func parseHostsFromString(text string) []string {
    text = strings.Replace(text, "\n", " ", -1)
    text = strings.Replace(text, "\t", " ", -1)
    text = strings.Replace(text, ",", " ", -1)
    raw := strings.Split(text, " ")
    var result []string
    for _, h := range raw {
        if h != " " && h != "" {
            result = append(result, h)
        }
    }
    return result
}
func showHelp() {
    fmt.Println("\033[33mUsage: sesh -f HOST_FILE -u USER -p PASSWORD COMMAND\033[0m")
    fmt.Println("\033[33mSimple usage(with config file ~/.seshrc): sesh -f HOST_FILE COMMAND\033[0m")
    flag.PrintDefaults()
}

func printDebugInfo(config map[string]interface{}, hosts []string) {
    fmt.Println("\033[33mConfigurations:\033[0m")
    fmt.Printf("\033[32mUser:\033[0m %v\n", config["User"])
    fmt.Printf("\033[32mKeyfile:\033[0m %v\n", config["Keyfile"])
    pas := ""
    if p, ok := config["Password"].(string); ok && p != "" {
        pas = "INVISIBLE"
    }
    fmt.Printf("\033[32mPassword:\033[0m %v\n", pas)
    if f, ok := config["Output"].(*os.File); ok {
        fmt.Printf("\033[32mOutput:\033[0m %v\n", f.Name())
    }
    fmt.Printf("\033[32mHosts:\033[0m %v\n", hosts)
    fmt.Printf("\033[32mComands:\033[0m\n%v\n", config["Cmd"])

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
