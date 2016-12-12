package main

import (
    "fmt"
    "strings"
)

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

func printDebugInfo(options SeshFlags, hosts []string, cmd string) {
    fmt.Println("\033[33mConfigurations:\033[0m")
    fmt.Printf("\033[32mUser:\033[0m %v\n", options.User)
    fmt.Printf("\033[32mKeyfile:\033[0m %v\n", options.Keyfile)
    fmt.Printf("\033[32mPassword:\033[0m %v\n", options.Password)
    fmt.Printf("\033[32mHosts:\033[0m %v\n", hosts)
    fmt.Printf("\033[32mComands:\033[0m\n%v\n", cmd)
    if options.Parallel {
        fmt.Printf("\033[32mParallel:\033[0m%v, degree: %v\n", options.Parallel, options.ParallelDegree)
        fmt.Printf("\033[32mtmp direcotry:\033[0m %v\n", options.Tmpdir)
    }
    fmt.Printf("\033[32mPause for check:\033[0m %v\n", options.Pause)
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
