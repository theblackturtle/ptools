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

    "golang.org/x/net/html"
)

var re = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml|cgi)(?:[\?|#][^"|']{0,}|)))(?:"|')`)
var Replacer = strings.NewReplacer(`\u003c`, `<`, `\u003e`, `>`, `\u0026`, `&`, `\u002f`, `/`, `\/`, `/`)

func main() {
    rawSource, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        panic(err)
    }
    contentType := http.DetectContentType(rawSource)
    rawSource = bytes.ToLower(rawSource)
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
    htmlToken := html.NewTokenizer(bytes.NewReader(source))
    for {
        // Next scans the next token and returns its type.
        tokenType := htmlToken.Next()
        // Token returns the next Token
        token := htmlToken.Token()
        switch {
        case tokenType == html.ErrorToken:
            return
        case tokenType == html.StartTagToken:
            switch token.Data {
            case "a":
                links = append(links, GetHref(token))
            case "img":
                links = append(links, GetSrc(token))
            case "script":
                links = append(links, GetSrc(token))
            case "link":
                links = append(links, GetHref(token))
            default:
            }
        case tokenType == html.TextToken || tokenType == html.CommentToken:
            text := html.UnescapeString(token.String())
            text = strings.ReplaceAll(text, "\\", `\`)
            text = Replacer.Replace(text)
            reLinks := regexExtract(text)
            links = append(links, reLinks...)
        }
    }
}

func parseOthers(source string) []string {
    links := make([]string, 0)
    source = strings.ReplaceAll(source, "\\", `\`)
    source = Replacer.Replace(source)
    reLinks := regexExtract(source)
    links = append(links, reLinks...)
    return links
}

// GetHref returns href values when present
func GetHref(t html.Token) (href string) {
    for _, a := range t.Attr {
        if (a.Key == "href" && a.Val != "#") || strings.Contains(a.Key, "href") {
            href = a.Val
        }
    }
    return
}

// GetSrc returns src values when present
func GetSrc(t html.Token) (src string) {
    for _, a := range t.Attr {
        if a.Key == "src" {
            src = a.Val
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
