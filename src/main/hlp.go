package main

import (
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
