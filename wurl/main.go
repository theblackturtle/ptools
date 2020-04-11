package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/valyala/fasthttp"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	UserAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36"
	Accept           = "*/*"
	AcceptLang       = "en-US,en;q=0.8"
	MaxBodySize      = 5242880
	MaxRedirectTimes = 16
)

var (
	client       *fasthttp.Client
	jsonOutput   bool
	redirect     bool
	threads      int
	timeout      time.Duration
	inputFile    string
	saveResponse bool
	outputFolder string
)

type Response struct {
	Url             string   `json:"url,omitempty"`
	IpAddress       string   `json:"ip_address,omitempty"`
	RedirectHistory []string `json:"redirect_history,omitempty"`
	StatusCode      int      `json:"status_code,omitempty"`
	ContentType     string   `json:"content_type,omitempty"`
	Size            int64    `json:"size,omitempty"`
	WordsSize       int64    `json:"words_size,omitempty"`
	LinesSize       int64    `json:"lines_size,omitempty"`
	Filename        string   `json:"file_name,omitempty"`
}

func main() {
	var timeoutInt int
	flag.StringVar(&inputFile, "i", "-", "Input file, default is Stdin")
	flag.BoolVar(&jsonOutput, "j", false, "Output as json")
	flag.BoolVar(&redirect, "r", false, "Enable redirect")
	flag.IntVar(&threads, "t", 10, "Threads to use")
	flag.IntVar(&timeoutInt, "k", 15, "Timeout (second)")
	flag.BoolVar(&saveResponse, "s", false, "Save response")
	flag.StringVar(&outputFolder, "o", "out", "Output folder")
	flag.Parse()

	if inputFile == "" {
		fmt.Fprintln(os.Stderr, "Please check your input again.")
		os.Exit(1)
	}

	timeout = time.Duration(timeoutInt) * time.Second
	client = &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, timeout)
		},
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient, // For "local error: tls: no renegotiation"
		},
		ReadBufferSize:      48*1024,
		WriteBufferSize:     48*1024,
		MaxConnsPerHost:     1024,
		MaxResponseBodySize: MaxBodySize,
	}

	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(threads, func(i interface{}) {
		defer wg.Done()
		u := i.(string)
		response, err := request(u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error: %s\n", u, err)
			return
		}

		if jsonOutput {
			if j, err := jsoniter.MarshalToString(response); err == nil {
				fmt.Println(j)
			}
		} else {
			if saveResponse {
				format := fmt.Sprintf("%s - [%d-%s] %s %d %d %d %v", response.Filename, response.StatusCode, strings.Split(response.ContentType, ";")[0], response.Url, response.Size, response.WordsSize, response.LinesSize, response.RedirectHistory)
				fmt.Println(format)
			} else {
				format := fmt.Sprintf("[%d-%s] %s %d %d %d %v", response.StatusCode, strings.Split(response.ContentType, ";")[0], response.Url, response.Size, response.WordsSize, response.LinesSize, response.RedirectHistory)
				fmt.Println(format)
			}
		}
	}, ants.WithPreAlloc(true))
	defer p.Release()

	var sc *bufio.Scanner
	if inputFile == "-" {
		sc = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(inputFile)
		if err != nil {
			panic(err)
		}
		sc = bufio.NewScanner(f)
	}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		u, err := url.Parse(line)
		if err == nil {
			wg.Add(1)
			p.Invoke(u.String())
		}
	}
	wg.Wait()
}

func request(u string) (Response, error) {
	var response Response
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", Accept)
	req.Header.Set("Accept-Language", AcceptLang)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	redirectsCount := 0
	for {
		req.SetRequestURI(u)
		err := client.DoTimeout(req, resp, timeout)
		if err != nil {
			if errors.Is(err, fasthttp.ErrBodyTooLarge) {
				return Response{}, nil
			} else {
				return response, errors.New(fmt.Sprintf("request error: %s", err))
			}
		}
		if fasthttp.StatusCodeIsRedirect(resp.StatusCode()) && redirect {
			redirectsCount++
			if redirectsCount > MaxRedirectTimes {
				return response, errors.New("too many redirects")
			}

			nextLocation := resp.Header.Peek(fasthttp.HeaderLocation)
			if len(nextLocation) == 0 {
				return response, errors.New("location header not found")
			}
			u = getRedirectURL(u, nextLocation)
			continue
		}
		break
	}

	contentType := string(resp.Header.Peek(fasthttp.HeaderContentType))
	if strings.Contains(contentType, "/html") && (resp.StatusCode() == 404 || resp.StatusCode() == 429) {
		return response, errors.New("404 found")
	}

	contentEncoding := string(resp.Header.Peek(fasthttp.HeaderContentEncoding))
	var body []byte
	var err error
	if contentEncoding == "gzip" {
		body, err = resp.BodyGunzip()
		if err != nil {
			return response, err
		}
	} else if contentEncoding == "deflate" {
		body, err = resp.BodyInflate()
		if err != nil {
			return response, err
		}
	} else {
		body = resp.Body()
	}

	var history []string
	// Get redirect history
	nextLocation := resp.Header.Peek(fasthttp.HeaderLocation)
	if len(nextLocation) > 0 {
		nextURL := getRedirectURL(u, nextLocation)
		history = append(history, nextURL)
	}
	bodyString := string(body)
	ipAddress := resp.RemoteAddr().String()

	response = Response{
		Url:             req.URI().String(),
		IpAddress:       ipAddress,
		RedirectHistory: history,
		StatusCode:      resp.StatusCode(),
		ContentType:     contentType,
		Size:            int64(utf8.RuneCountInString(bodyString)),
		WordsSize:       int64(len(strings.Split(bodyString, " "))),
		LinesSize:       int64(len(strings.Split(bodyString, "\n"))),
	}
	if saveResponse {
		savePath := save(bodyString, req, resp)
		if savePath != "" {
			response.Filename = savePath
		}
	}
	return response, nil
}

func save(bodyString string, req *fasthttp.Request, resp *fasthttp.Response) string {
	hash := sha1.Sum([]byte(req.URI().String()))
	respPath := path.Join(outputFolder, string(req.URI().Host()), fmt.Sprintf("%x", hash))
	err := os.MkdirAll(path.Dir(respPath), 0750)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create new directory: %s\n", err)
		return ""
	}

	respFile, err := os.Create(respPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create new file: %s\n", err)
		return ""
	}
	buf := &bytes.Buffer{}
	buf.WriteString(req.URI().String())
	buf.WriteString("\n\n")

	// request headers
	req.Header.VisitAll(func(key, value []byte) {
		buf.WriteString(fmt.Sprintf("> %s: %s\n", string(key), string(value)))
	})

	buf.WriteString("\n")

	// TODO: Request time
	//buf.WriteString(fmt.Sprintf("< Request Time: %d secs\n", resp.ReceivedAt().Second()))

	// Response headers
	buf.WriteString(fmt.Sprintf("< HTTP/1.1 %d\n", resp.StatusCode()))
	resp.Header.VisitAll(func(key, value []byte) {
		buf.WriteString(fmt.Sprintf("< %s: %s\n", string(key), string(value)))
	})

	buf.WriteString("\n")
	buf.WriteString(bodyString)

	respFile.WriteString(buf.String())
	respFile.Close()
	return respPath
}

func getRedirectURL(baseURL string, location []byte) string {
	u := fasthttp.AcquireURI()
	u.Update(baseURL)
	u.UpdateBytes(location)
	redirectURL := u.String()
	fasthttp.ReleaseURI(u)
	return redirectURL
}
