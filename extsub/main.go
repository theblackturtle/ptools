package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "regexp"
    "strconv"
    "strings"

    "github.com/emirpasic/gods/sets/treeset"
    "golang.org/x/net/publicsuffix"
)

// SUBRE is a regular expression that will match on all subdomains once the domain is appended.
const SUBRE = "(([a-zA-Z0-9]{1}|[_a-zA-Z0-9]{1}[_a-zA-Z0-9-]{0,61}[a-zA-Z0-9]{1})[.]{1})+"

var (
    anySubRE    = AnySubdomainRegex()
    nameStripRE = regexp.MustCompile(`^u[0-9a-f]{4}|20|22|25|2b|2f|3d|3a|40`)
)

func main() {
    var domain string
    flag.StringVar(&domain, "d", "", "Main domain")
    flag.Parse()
    text, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        return
    }
    set := treeset.NewWithStringComparator()
    if domain == "" {
        for _, name := range anySubRE.FindAllString(string(text), -1) {
            name = cleanName(name)
            if _, valid := publicsuffix.PublicSuffix(name); valid {
                name = reverse(name)
                if !set.Contains(name) {
                    set.Add(name)
                }
            }
        }
    } else {
        subRE := SubdomainRegex(domain)
        for _, name := range subRE.FindAllString(string(text), -1) {
            name = cleanName(name)
            name = reverse(name)
            if !set.Contains(name) {
                set.Add(name)
            }
        }
    }
    set.Each(func(_ int, value interface{}) {
        fmt.Println(reverse(value.(string)))
    })
    fmt.Println("============= DONE =============")
}

// SubdomainRegex returns a Regexp object initialized to match
// subdomain names that end with the domain provided by the parameter.
func SubdomainRegex(domain string) *regexp.Regexp {
    // Change all the periods into literal periods for the regex
    d := strings.Replace(domain, ".", "[.]", -1)
    return regexp.MustCompile(SUBRE + d)
}

// AnySubdomainRegex returns a Regexp object initialized to match any DNS subdomain name.
func AnySubdomainRegex() *regexp.Regexp {
    return regexp.MustCompile(SUBRE + "[a-zA-Z]{2,61}")
}

// Clean up the names scraped from the web.
func cleanName(name string) string {
    var err error

    name, err = strconv.Unquote("\"" + strings.TrimSpace(name) + "\"")
    if err == nil {
        name = anySubRE.FindString(name)
    }

    name = strings.ToLower(name)
    for {
        name = strings.Trim(name, "-.")
        if i := nameStripRE.FindStringIndex(name); i != nil {
            name = name[i[1]:]
        } else {
            break
        }
    }
    return name
}

func reverse(s string) string {
    b := make([]byte, len(s))
    var j = len(s) - 1
    for i := 0; i <= j; i++ {
        b[j-i] = s[i]
    }

    return string(b)
}
