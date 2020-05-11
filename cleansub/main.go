package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
)

var nameStripRE = regexp.MustCompile(`^u[0-9a-f]{4}|20|22|25|2b|2f|3d|3a|40`)
var subdomainRE = regexp.MustCompile(`(([a-zA-Z0-9]{1}|[_a-zA-Z0-9]{1}[_a-zA-Z0-9-]{0,61}[a-zA-Z0-9]{1})[.]{1})+[a-zA-Z]{2,61}`)

func main() {
    sc := bufio.NewScanner(os.Stdin)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" {
            continue
        }
        name := subdomainRE.FindString(line)
        name = strings.ToLower(name)
        for {
            name = strings.Trim(name, "-.")
            if i := nameStripRE.FindStringIndex(name); i != nil {
                name = name[i[1]:]
            } else {
                break
            }
        }
        name = removeAsteriskLabel(name)
        fmt.Println(name)
    }
}

func removeAsteriskLabel(s string) string {
    startIndex := strings.LastIndex(s, "*.")

    if startIndex == -1 {
        return s
    }

    return s[startIndex+2:]
}
