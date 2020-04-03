package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type JsonResult struct {
	Input            map[string]string `json:"input"`
	Position         int               `json:"position"`
	StatusCode       int64             `json:"status"`
	ContentLength    int64             `json:"length"`
	ContentWords     int64             `json:"words"`
	ContentLines     int64             `json:"lines"`
	RedirectLocation string            `json:"redirectlocation"`
	ResultFile       string            `json:"resultfile"`
	Url              string            `json:"url"`
}

type jsonFileOutput struct {
	CommandLine string       `json:"commandline"`
	Time        string       `json:"time"`
	Results     []JsonResult `json:"results"`
}

func main() {
	var limit int
	flag.IntVar(&limit, "l", 50, "Limit")
	flag.Parse()
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file")
		os.Exit(1)
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file")
		os.Exit(1)
	}
	foundUrls := &jsonFileOutput{}
	err = jsoniter.Unmarshal(data, foundUrls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to Unmarshal: %s", err)
		os.Exit(1)
	}

	resultsMap := make(map[string][]JsonResult)
	for _, result := range foundUrls.Results {
		u, err := url.Parse(result.Url)
		if err != nil {
			continue
		}
		hostname := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

		if rList := resultsMap[hostname]; len(rList) == 0 {
			resultsMap[hostname] = []JsonResult{result}
		} else {
			resultsMap[hostname] = append(resultsMap[hostname], result)
		}
	}

	var results []JsonResult
	for _, foundList := range resultsMap {
		p := mostStatusCode(foundList)
		if len(p) == 0 {
			continue
		}
		// len(blackList)) == 0 mean get all
		var blackList []int64
		for statusCode, times := range p {
			if times >= limit {
				blackList = append(blackList, statusCode)
			}
		}

		for _, found := range foundList {
			if !inSlice(blackList, found.StatusCode) {
				results = append(results, found)
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		si_url, err := url.Parse(results[i].Url)
		if err != nil {
			return true
		}
		sj_url, err := url.Parse(results[j].Url)
		if err != nil {
			return true
		}

		si_lower := strings.ToLower(si_url.Hostname())
		sj_lower := strings.ToLower(sj_url.Hostname())
		if si_lower < sj_lower {
			return true
		} else {
			return false
		}
	})

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].StatusCode < results[j].StatusCode {
			return true
		} else {
			return false
		}
	})

	for _, r := range results {
		f := fmt.Sprintf("%s,%d,%d,%d,%d,%s", r.Url, r.StatusCode, r.ContentLength, r.ContentWords, r.ContentLines, r.RedirectLocation)
		fmt.Println(f)
	}
}

func mostStatusCode(slice []JsonResult) map[int64]int {
	m := make(map[int64]int)
	for _, r := range slice {
		if _, ok := m[r.StatusCode]; !ok {
			m[r.StatusCode] = 1
		} else {
			m[r.StatusCode]++
		}
	}
	return m
}

func inSlice(slice []int64, s int64) bool {
	for _, e := range slice {
		if s == e {
			return true
		}
	}
	return false
}
