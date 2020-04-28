package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "regexp"
    "strconv"
    "strings"
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
        panic(err)
    }
    if domain == "" {
        for _, name := range anySubRE.FindAllString(string(text), -1) {
            name = cleanName(name)
            fmt.Println(name)
        }
    } else {
        subRE := SubdomainRegex(domain)
        for _, name := range subRE.FindAllString(string(text), -1) {
            name = cleanName(name)
            fmt.Println(name)
        }
    }
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
