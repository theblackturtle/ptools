package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
)

var nameStripRE = regexp.MustCompile(`^u[0-9a-f]{4}|20|22|25|2b|2f|3d|3a|40`)
var chars = []string{
    `-.`,
    `*.`,
    `*-`,
    `"`,
    `'`,
    `(`,
    `)`,
    `.`,
}

func main() {
    sc := bufio.NewScanner(os.Stdin)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" {
            continue
        }
        name := strings.ToLower(line)
        name = trimChars(name)
        if i := nameStripRE.FindStringIndex(name); i != nil {
            name = name[i[1]:]
        }
        fmt.Println(name)
    }
}

func trimChars(name string) string {
    for _, c := range chars {
        name = strings.Trim(name, c)
    }
    return name
}
