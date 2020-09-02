package main

import (
    "bytes"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "sort"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "github.com/ditashi/jsbeautifier-go/jsbeautifier"
    "golang.org/x/net/html"
)

var linkfinderRegex = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml|cgi)(?:[\?|#][^"|']{0,}|)))(?:"|')`)
var Replacer = strings.NewReplacer(
    `\u003c`, `<`,
    `\U003C`, `<`,
    `\u003C`, `<`,
    `\U003c`, `<`,

    `\u003e`, `>`,
    `\U003E`, `>`,
    `\u003E`, `>`,
    `\U003e`, `>`,

    `\u0026`, `&`,
    `\U0026`, `&`,

    `\u002f`, `/`,
    `\U002F`, `/`,
    `\u002F`, `/`,
    `\U002f`, `/`,

    `\/`, `/`,
    `\\`, `\`,
)
var jsBeautifyOptions = map[string]interface{}{
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
    var printDone bool
    flag.BoolVar(&printDone, "p", false, "Print DONE when finish")
    flag.Parse()
    rawSource, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        panic(err)
    }
    contentType := http.DetectContentType(rawSource)
    rawSourceStr := string(rawSource)
    rawSourceStr = html.UnescapeString(html.UnescapeString(rawSourceStr))

    rawSource = []byte(rawSourceStr)

    var links []string
    if strings.Contains(contentType, "html") {
        links = parseHTML(rawSource)
    } else if strings.Contains(contentType, "text") {
        links = parseOthers(rawSourceStr)
    } else {
        fmt.Println("Not support this content type yet")
        os.Exit(0)
    }

    uniqueLinks := unique(links)
    sort.Strings(uniqueLinks)
    for _, e := range uniqueLinks {
        if len(e) == 0 {
            continue
        }
        fmt.Println(e)
    }
    if printDone {
        fmt.Println("============= DONE =============")
    }
}

func parseHTML(source []byte) (links []string) {
    links = make([]string, 0)
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(source))
    if err != nil {
        return
    }
    doc.Find("img").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetSrc(n)...)

    })
    doc.Find("script").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetSrc(n)...)
        scriptText := selection.Text()
        links = append(links, parseOthers(scriptText)...)
    })

    doc.Find("a").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetHref(n)...)
    })
    doc.Find("link").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetHref(n)...)
    })
    return
}

func parseOthers(source string) []string {
    links := make([]string, 0)
    source = Replacer.Replace(source)
    source = html.UnescapeString(source)
    reLinks := regexExtract(source)
    links = append(links, reLinks...)
    return links
}

// GetHref returns href values when present
func GetHref(t *html.Node) (hrefs []string) {
    hrefs = make([]string, 0)
    for _, a := range t.Attr {
        if strings.Contains(a.Key, "href") && a.Val != "#" {
            hrefs = append(hrefs, a.Val)
        }
    }
    return
}

// GetSrc returns src values when present
func GetSrc(t *html.Node) (srcs []string) {
    srcs = make([]string, 0)
    for _, a := range t.Attr {
        if strings.Contains(a.Key, "src") {
            srcs = append(srcs, a.Val)
        }
    }
    return
}

func filterNewLines(s string) string {
    return regexp.MustCompile(`[\t\r\n]+`).ReplaceAllString(strings.TrimSpace(s), " ")
}

func regexExtract(source string) []string {
    var links []string
    source, _ = jsbeautifier.Beautify(&source, jsBeautifyOptions)

    matches := linkfinderRegex.FindAllStringSubmatch(source, -1)
    for _, match := range matches {
        matchGroup1 := filterNewLines(match[1])
        if matchGroup1 == "" {
            continue
        }
        link := strings.Trim(matchGroup1, `\`)
        link = html.UnescapeString(link)
        links = append(links, link)
    }
    return links
}

func unique(elements []string) []string {
    seen := map[string]bool{}
    var results []string

    for _, e := range elements {
        e = strings.TrimSpace(e)
        if _, ok := seen[e]; !ok {
            seen[e] = true
            results = append(results, e)
        }
    }
    return results
}
