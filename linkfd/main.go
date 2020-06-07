package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "regexp"
    "sort"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "golang.org/x/net/html"
)

var re = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml|cgi)(?:[\?|#][^"|']{0,}|)))(?:"|')`)
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

func main() {
    rawSource, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        panic(err)
    }
    contentType := http.DetectContentType(rawSource)
    rawSource = []byte(html.UnescapeString(string(rawSource)))

    var links []string
    if strings.Contains(contentType, "html") {
        links = parseHTML(rawSource)
    } else if strings.Contains(contentType, "text") {
        links = parseOthers(string(rawSource))
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
    fmt.Println("============= DONE =============")
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

    })
    doc.Find("a").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetHref(n)...)
    })
    doc.Find("link").Each(func(i int, selection *goquery.Selection) {
        n := selection.Get(0)
        links = append(links, GetHref(n)...)
    })
    // doc.Contents().Each(func(i int, selection *goquery.Selection) {
    //     // fmt.Println(selection.Text())
    //     if goquery.NodeName(selection) == "#text" {
    //         fmt.Println(selection.Text())
    //     }
    // })
    //         case tokenType == html.TextToken || tokenType == html.CommentToken:
    //             text := html.UnescapeString(token.String())
    //             text = strings.ReplaceAll(text, "\\", `\`)
    //             text = Replacer.Replace(text)
    //             reLinks := regexExtract(text)
    //             links = append(links, reLinks...)

    return
}

func parseOthers(source string) []string {
    links := make([]string, 0)
    source = Replacer.Replace(source)
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
    matches := re.FindAllStringSubmatch(source, -1)
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
