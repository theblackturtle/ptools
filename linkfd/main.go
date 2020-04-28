package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "regexp"
    "sort"
    "strings"

    "github.com/ditashi/jsbeautifier-go/jsbeautifier"
)

var re = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`)

var options = map[string]interface{}{
    "indent_size":               4,
    "indent_char":               " ",
    "indent_with_tabs":          false,
    "preserve_newlines":         true,
    "max_preserve_newlines":     10,
    "space_in_paren":            false,
    "space_in_empty_paren":      false,
    "e4x":                       false,
    "jslint_happy":              false,
    "space_after_anon_function": false,
    "brace_style":               "collapse",
    "keep_array_indentation":    false,
    "keep_function_indentation": false,
    "eval_code":                 false,
    "unescape_strings":          true,
    "wrap_line_length":          0,
    "break_chained_methods":     false,
    "end_with_newline":          false,
}

func main() {
    rawSource, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        panic(err)
    }
    sRawSource := string(rawSource)

    var beautifySource string
    if len(sRawSource) > 1000000 {
        beautifySource = strings.ReplaceAll(sRawSource, ";", ";\r\n")
        beautifySource = strings.ReplaceAll(sRawSource, ",", ",\r\n")
    } else {
        beautifySource, err = jsbeautifier.Beautify(&sRawSource, options)
        if err != nil {
            fmt.Println("Failed to beautify")
            os.Exit(1)
        }
    }
    var links []string
    match := re.FindAllStringSubmatch(beautifySource, -1)
    for _, m := range match {
        matchGroup1 := filterNewLines(m[1])
        if matchGroup1 == "" {
            continue
        }
        links = append(links, matchGroup1)
    }
    uniqueLinks := unique(links)
    sort.Strings(uniqueLinks)
    for _, e := range uniqueLinks {
        fmt.Println(e)
    }

}

func filterNewLines(s string) string {
    return regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(s), " ")
}

func unique(s []string) []string {
    seen := map[string]bool{}
    uniqSlice := []string{}
    for _, e := range s {
        if _, ok := seen[e]; !ok {
            seen[e] = true
            uniqSlice = append(uniqSlice, e)
        }
    }
    return uniqSlice
}
