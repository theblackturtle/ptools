package main

import (
    "bufio"
    "bytes"
    "crypto/sha1"
    "crypto/tls"
    "errors"
    "flag"
    "fmt"
    "html"
    "net"
    "net/http"
    "net/url"
    "os"
    "path"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"
    "unicode/utf8"

    jsoniter "github.com/json-iterator/go"
    "github.com/panjf2000/ants/v2"
    "github.com/valyala/fasthttp"
)

const (
    UserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36"
    Accept      = "*/*"
    AcceptLang  = "en-US,en;q=0.8"
    MaxBodySize = 5242880
    MaxRedirect = 16
)

var (
    Client *fasthttp.Client
    Pool   *ants.PoolWithFunc
    wg     sync.WaitGroup

    titleRegex = regexp.MustCompile(`<[Tt][Ii][Tt][Ll][Ee][^>]*>([^<]*)</[Tt][Ii][Tt][Ll][Ee]>`)
    // wordRegex  = regexp.MustCompile(`[^.a-zA-Z0-9_-]`)
    headerList = make(headerArgs, 0)

    jsonOutput    bool
    redirect      bool
    threads       int
    timeout       time.Duration
    inputFile     string
    saveResponse  bool
    outputFolder  string
    ignore404Html bool
    debug         bool
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
    RequestTime     string
}

type Task struct {
    Url          string
    RedirectTime int
}

type headerArgs map[string]string

func (h headerArgs) String() string {
    return ""
}

func (h *headerArgs) Set(val string) error {
    args := strings.SplitN(val, ":", 2)
    (*h)[strings.TrimSpace(args[0])] = strings.TrimSpace(args[1])
    return nil
}

func main() {
    var timeoutInt int
    flag.StringVar(&inputFile, "i", "-", "Input file, default is Stdin")
    flag.BoolVar(&jsonOutput, "j", false, "Output as json")
    flag.BoolVar(&redirect, "r", false, "Enable redirect")
    flag.IntVar(&threads, "t", 40, "Threads to use")
    flag.IntVar(&timeoutInt, "k", 20, "Timeout (second)")
    flag.BoolVar(&saveResponse, "s", false, "Save response")
    flag.StringVar(&outputFolder, "o", "out", "Output folder")
    flag.BoolVar(&ignore404Html, "x", false, "Ignore HTML response with 404 and 429 status code")
    flag.BoolVar(&debug, "d", false, "Enable debug")
    flag.Var(&headerList, "H", "Header to request, can set multiple (Host: localhost)")
    flag.Parse()

    if inputFile == "" {
        fmt.Fprintln(os.Stderr, "Please check your input again.")
        os.Exit(1)
    }

    timeout = time.Duration(timeoutInt) * time.Second
    Client = &fasthttp.Client{
        NoDefaultUserAgentHeader: true,
        Dial: func(addr string) (net.Conn, error) {
            return fasthttp.DialDualStackTimeout(addr, time.Second*30)
        },
        TLSConfig: &tls.Config{
            InsecureSkipVerify: true,
            Renegotiation:      tls.RenegotiateOnceAsClient, // For "local error: tls: no renegotiation"
        },
        ReadBufferSize:         48 << 10, // 48KB
        WriteBufferSize:        48 << 10,
        MaxConnsPerHost:        1024,
        MaxResponseBodySize:    MaxBodySize,
        DisablePathNormalizing: true,
    }

    var indexFile *os.File
    if saveResponse {
        err := os.MkdirAll(outputFolder, 0755)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to create new directory: %s\n", err)
            os.Exit(1)
        }
        indexFile, err = os.OpenFile(path.Join(outputFolder, "index"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Failed to create index file: %s\n", err)
            os.Exit(1)
        }
        defer indexFile.Close()
    }

    worker := func(i interface{}) {
        defer wg.Done()
        task := i.(Task)
        if task.RedirectTime > MaxRedirect {
            if debug {
                fmt.Fprintf(os.Stderr, "%s redirect more than 16 times\n", task.Url)
            }
            return
        }

        response, err := Request(task)
        if err != nil {
            if debug {
                fmt.Fprintf(os.Stderr, "%s error: %s\n", task.Url, err)
            }
            return
        }

        if jsonOutput {
            if j, err := jsoniter.MarshalToString(response); err == nil {
                fmt.Println(j)
            }
        } else {
            if saveResponse {
                format := fmt.Sprintf("%s [%d-%s] %s %d %d %d %v", response.Filename, response.StatusCode, strings.Split(response.ContentType, ";")[0], response.Url, response.Size, response.WordsSize, response.LinesSize, response.RedirectHistory)
                indexFile.WriteString(format + "\n")
            } else {
                format := fmt.Sprintf("[%d-%s] %s %d %d %d %v", response.StatusCode, strings.Split(response.ContentType, ";")[0], response.Url, response.Size, response.WordsSize, response.LinesSize, response.RedirectHistory)
                fmt.Println(format)
            }
        }
    }

    Pool, _ = ants.NewPoolWithFunc(threads, worker, ants.WithPreAlloc(true))
    defer Pool.Release()

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
        if u, err := url.Parse(line); err == nil {
            task := Task{
                Url:          u.String(),
                RedirectTime: 0,
            }
            wg.Add(1)
            Pool.Invoke(task)
        }
    }
    wg.Wait()
}

func Request(task Task) (Response, error) {
    var response Response
    var elapsed string
    req := fasthttp.AcquireRequest()
    defer fasthttp.ReleaseRequest(req)
    req.Header.Set("User-Agent", UserAgent)
    req.Header.Set("Accept", Accept)
    req.Header.Set("Accept-Language", AcceptLang)
    if len(headerList) > 0 {
        for key, value := range headerList {
            req.Header.Set(key, value)
        }
    }

    resp := fasthttp.AcquireResponse()
    defer fasthttp.ReleaseResponse(resp)

    var history []string

    // req.SetRequestURI(u)
    req.Header.SetRequestURI(task.Url)

    start := time.Now()
    err := Client.DoTimeout(req, resp, timeout)
    if err != nil {
        if errors.Is(err, fasthttp.ErrBodyTooLarge) {
            return Response{}, nil
        } else {
            return response, fmt.Errorf("Request error: %s", err)
        }
    }
    elapsed = time.Since(start).String()
    if fasthttp.StatusCodeIsRedirect(resp.StatusCode()) {
        nextLocation := resp.Header.Peek(fasthttp.HeaderLocation)
        if len(nextLocation) == 0 {
            return response, fmt.Errorf("location header not found")
        }
        nextUrl := getRedirectURL(task.Url, nextLocation)
        history = append(history, nextUrl)
        if redirect || justRedirectToHTTPS(task.Url, nextUrl) {
            wg.Add(1)
            newTask := Task{
                Url:          nextUrl,
                RedirectTime: task.RedirectTime + 1,
            }
            Pool.Invoke(newTask) // Add it direct to pool, we can get all of redirects
        }
    }

    contentType := string(resp.Header.Peek(fasthttp.HeaderContentType))
    if ignore404Html {
        if strings.Contains(contentType, "html") && (resp.StatusCode() == 404 || resp.StatusCode() == 429) {
            return response, fmt.Errorf("%d found", resp.StatusCode())
        }
    }

    contentEncoding := string(resp.Header.Peek(fasthttp.HeaderContentEncoding))
    var body []byte
    switch contentEncoding {
    case "gzip":
        body, err = resp.BodyGunzip()
        if err != nil {
            return response, err
        }
    case "deflate":
        body, err = resp.BodyInflate()
        if err != nil {
            return response, err
        }
    default:
        body = resp.Body()
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
        // WordsSize:       len(wordRegex.FindAllString(bodyString, -1)),
        WordsSize:   int64(len(strings.Split(bodyString, " "))),
        LinesSize:   int64(len(strings.Split(bodyString, "\n"))),
        RequestTime: elapsed,
    }
    if saveResponse {
        savePath := save(bodyString, req, resp, response)
        if savePath != "" {
            response.Filename = savePath
        }
    }
    return response, nil
}

func save(bodyString string, req *fasthttp.Request, resp *fasthttp.Response, r Response) string {
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

    // Summary
    title := titleRegex.FindStringSubmatch(bodyString)
    if len(title) > 0 {
        buf.WriteString("# Title: " + strings.TrimSpace(html.UnescapeString(title[1])))
        buf.WriteString("\n")
    }
    nextLocation := string(resp.Header.Peek("Location"))
    if nextLocation != "" {
        buf.WriteString("# Location: " + nextLocation)
        buf.WriteString("\n")
    }
    buf.WriteString("# Size: " + strconv.FormatInt(r.Size, 10))
    buf.WriteString("\n")
    buf.WriteString("# Words: " + strconv.FormatInt(r.WordsSize, 10))
    buf.WriteString("\n")
    buf.WriteString("# Lines: " + strconv.FormatInt(r.LinesSize, 10))
    buf.WriteString("\n")
    buf.WriteString("# IP: " + r.IpAddress)
    buf.WriteString("\n")
    buf.WriteString("# Request Time: " + r.RequestTime)

    buf.WriteString("\n\n")

    // Request headers
    req.Header.VisitAll(func(key, value []byte) {
        buf.WriteString(fmt.Sprintf("> %s: %s\n", string(key), string(value)))
    })

    buf.WriteString("\n")

    // Response headers
    buf.WriteString(fmt.Sprintf("< HTTP/1.1 %d %s\n", resp.StatusCode(), http.StatusText(resp.StatusCode())))
    resp.Header.VisitAll(func(key, value []byte) {
        buf.WriteString(fmt.Sprintf("< %s: %s\n", string(key), string(value)))
    })

    // buf.WriteString("\n")
    buf.Write([]byte("\r\n"))

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

func justRedirectToHTTPS(originalUrl, redirectUrl string) bool {
    oUrl, err := url.Parse(originalUrl)
    if err != nil {
        return false
    }
    rUrl, err := url.Parse(redirectUrl)
    if err != nil {
        return false
    }
    if oUrl.Hostname() == rUrl.Hostname() && oUrl.EscapedPath() == rUrl.EscapedPath() && oUrl.RawQuery == rUrl.RawQuery {
        return true
    }
    return false
}
