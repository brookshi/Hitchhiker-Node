package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type testCase struct {
	RequestBodyList  []requestBody     `json:"requestBodyList"`
	EnvVariables     map[string]string `json:"envVariables"`
	Repeat           int               `json:"repeat"`
	ConcurrencyCount int               `json:"concurrencyCount"`
	QPS              int               `json:"qps"`
	Timeout          int               `json:"timeout"`
	KeepAlive        bool              `json:"keepAlive"`
	results          chan []runResult
	forceStop        int32
	trace            func(rst runResult)
}

type requestBody struct {
	ID      string            `json:"id"`
	Param   string            `json:"param"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Tests   string            `json:"test"`
}

type runResult struct {
	ID            string            `json:"id"`
	Success       bool              `json:"success"`
	Param         string            `json:"param"`
	Err           runError          `json:"error"`
	Body          string            `json:"body"`
	Status        int               `json:"status"`
	StatusMessage string            `json:"statusMessage"`
	Duration      duration          `json:"duration"`
	Headers       map[string]string `json:"headers"`
	Tests         map[string]bool   `json:"tests"`
}

type duration struct {
	DNS     time.Duration `json:"dns"`
	Connect time.Duration `json:"connect"`
	Request time.Duration `json:"request"`
}

type runError struct {
	Message string `json:"message"`
}

func (c *testCase) Run() {
	c.results = make(chan []runResult, c.Repeat*c.ConcurrencyCount)
	atomic.StoreInt32(&c.forceStop, 0)
	c.start()
	close(c.results)
}

func (c *testCase) start() {
	var waiter sync.WaitGroup
	waiter.Add(c.ConcurrencyCount)

	for i := 0; i < c.ConcurrencyCount; i++ {
		go func() {
			c.work(c.Repeat)
			waiter.Done()
		}()
	}
	waiter.Wait()
}

func (c *testCase) work(times int) {
	var throttle <-chan time.Time
	if c.QPS > 0 {
		throttle = time.Tick(time.Duration(1e6/(c.QPS)) * time.Microsecond)
	}

	transport := &http.Transport{
		DisableKeepAlives: !c.KeepAlive,
	}
	httpClient := http.Client{Transport: transport, Timeout: time.Duration(c.Timeout) * time.Second}
	for i := 0; i < times; i++ {
		if c.QPS > 0 {
			<-throttle
		}
		if atomic.CompareAndSwapInt32(&c.forceStop, 1, 1) {
			break
		}
		c.doRequest(httpClient)
	}
}

func (c *testCase) doRequest(httpClient http.Client) {
	results := make([]runResult, len(c.RequestBodyList))
	variables := make(map[string]string)
	cookies := make(map[string]string)
	for i, body := range c.RequestBodyList {
		results[i] = doRequestItem(body, httpClient, c.EnvVariables, variables, cookies)
		if c.trace != nil {
			c.trace(results[i])
		}
	}
	c.results <- results
}

func (c *testCase) stop() {
	atomic.CompareAndSwapInt32(&c.forceStop, 0, 1)
}

func doRequestItem(body requestBody, httpClient http.Client, envVariables map[string]string, variables map[string]string, cookies map[string]string) (result runResult) {
	var dnsStart, connectStart, reqStart time.Time
	var duration duration
	result = runResult{ID: body.ID, Param: body.Param, Success: false}

	//now := time.Now()
	req, err := buildRequest(body, cookies, envVariables, variables)

	if err != nil {
		fmt.Println(err)
		result.Err = runError{err.Error()}
		if req != nil {
			req.Close = true
		}
	} else {
		trace := &httptrace.ClientTrace{
			DNSStart: func(info httptrace.DNSStartInfo) {
				dnsStart = time.Now()
			},
			DNSDone: func(info httptrace.DNSDoneInfo) {
				duration.DNS = time.Now().Sub(dnsStart)
			},
			GetConn: func(hostPort string) {
				connectStart = time.Now()
			},
			GotConn: func(info httptrace.GotConnInfo) {
				duration.Connect = time.Now().Sub(connectStart)
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				reqStart = time.Now()
			},
			GotFirstResponseByte: func() {
				duration.Request = time.Now().Sub(reqStart)
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		res, err := httpClient.Do(req)
		if err == nil {
			result.Duration = duration
			result.Status = res.StatusCode
			result.StatusMessage = parseStatusMsg(res.Status, res.StatusCode)
			result.Headers = make(map[string]string)
			for k, v := range res.Header {
				hv := strings.Join(v, ";")
				result.Headers[k] = hv
				if strings.ToLower(k) == "cookie" {
					for sk, sv := range readCookies(hv) {
						cookies[sk] = sv
					}
				}
			}
			content, err := ioutil.ReadAll(res.Body)
			if err != nil {
				result.Err = runError{err.Error()}
			} else {
				result.Body = string(content)
			}

			testsStr := prepareTests(body.Tests, variables, envVariables)
			testResult := interpret(Otto, testsStr, result)
			result.Tests = make(map[string]bool)
			for k, v := range testResult.Tests {
				result.Tests[k] = v
			}
			for k, v := range testResult.Variables {
				variables[k] = v
			}
			result.Success = true
			defer res.Body.Close()
		} else {
			result.Err = runError{err.Error()}
		}
	}
	return
}

func buildRequest(reqBody requestBody, cookies map[string]string, envVariables map[string]string, variables map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if reqBody.Body != "" {
		bodyReader = strings.NewReader(applyAllVariables(reqBody.Body, variables, envVariables))
	}
	req, err := http.NewRequest(reqBody.Method, encodeUrl(applyAllVariables(reqBody.URL, variables, envVariables)), bodyReader)
	if err != nil {
		return nil, err
	}
	headers := make(http.Header)
	for k, v := range reqBody.Headers {
		if strings.ToLower(k) == "cookie" {
			v = applyCookies(v, cookies)
		}
		headers.Set(applyAllVariables(k, variables, envVariables), applyAllVariables(v, variables, envVariables))
	}
	req.Header = headers
	return req, nil
}

func applyAllVariables(content string, variables map[string]string, envVariables map[string]string) string {
	return applyVariables(applyVariables(content, variables), envVariables)
}

func applyVariables(content string, variables map[string]string) string {
	if len(content) == 0 || len(variables) == 0 {
		return content
	}
	for k, v := range variables {
		content = strings.Replace(content, fmt.Sprintf("{{%s}}", k), v, -1)
	}
	return content
}

func applyCookies(value string, cookies map[string]string) string {
	if len(cookies) == 0 || value == "nocookie" {
		return value
	}

	recordCookies := readCookies(value)
	for k, v := range cookies {
		if _, ok := recordCookies[k]; !ok {
			value = fmt.Sprintf("%s;%s", value, v)
		}
	}
	return value
}

func readCookies(cookies string) map[string]string {
	cookieMap := make(map[string]string)
	cookieArr := strings.Split(cookies, ";")
	for _, v := range cookieArr {
		v = strings.Trim(v, " ")
		i := strings.Index(v, "=")
		if i < 0 {
			i = len(v)
		}
		cookieMap[v[:i]] = v
	}
	return cookieMap
}

func prepareTests(tests string, variables map[string]string, envVariables map[string]string) string {
	return applyAllVariables(tests, variables, envVariables)
}

func parseStatusMsg(status string, statusCode int) string {
	statusCodeStr := string(statusCode)
	if strings.HasPrefix(status, statusCodeStr) {
		status = strings.Trim(strings.TrimPrefix(status, statusCodeStr), " ")
	}
	return status
}

func encodeUrl(u string) string {
	var Url *url.URL
	Url, err := url.Parse(u)
	if err != nil {
		return u
	}

	queryStrings := Url.Query()
	for k, v := range queryStrings {
		queryStrings[k] = []string{url.QueryEscape(v[0])}
	}
	Url.RawQuery = queryStrings.Encode()
	return Url.String()
}
