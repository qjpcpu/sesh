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

func parseData(datas []string) map[string]interface{} {
	kv := make(map[string]interface{})
	for _, data := range datas {
		data = strings.TrimSpace(data)
		if data == "" || !strings.Contains(data, "=") {
			continue
		}
		arr := strings.SplitN(data, "=", 2)
		kv[arr[0]] = arr[1]
	}
	return kv
}
