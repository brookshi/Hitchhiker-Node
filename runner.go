package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"
)

type testCase struct {
	requestBodys     []requestBody
	totalCount       int
	concurrencyCount int
	qps              int
	timeout          int
	results          chan []runResult
	trace            func(rst runResult)
}

type requestItem struct {
	body   requestBody
	result runResult
}

type requestBody struct {
	id      string
	method  string
	url     string
	body    string
	headers map[string]string
	tests   string
}

type runResult struct {
	id            string
	err           runError
	body          string
	status        int
	statusMessage string
	duration      duration
	headers       http.Header
	tests         map[string]bool
}

type duration struct {
	dns     time.Duration
	connect time.Duration
	request time.Duration
}

type runError struct {
	message string
}

func (c *testCase) Run() {
	c.results = make(chan []runResult, c.totalCount)
	c.start()
	close(c.results)
}

func (c *testCase) start() {
	var waiter sync.WaitGroup
	waiter.Add(c.concurrencyCount)

	for i := 0; i < c.concurrencyCount; i++ {
		go func() {
			c.work(c.totalCount / c.concurrencyCount)
			waiter.Done()
		}()
	}
	waiter.Wait()
}

func (c *testCase) work(times int) {
	var throttle <-chan time.Time
	if c.qps > 0 {
		throttle = time.Tick(time.Duration(1e6/(c.qps)) * time.Microsecond)
	}

	httpClient := http.Client{Timeout: time.Duration(c.timeout) * time.Second}
	for i := 0; i < times; i++ {
		if c.qps > 0 {
			<-throttle
		}
		c.doRequest(httpClient)
	}
}

func (c *testCase) doRequest(httpClient http.Client) {
	results := make([]runResult, len(c.requestBodys))
	for i, body := range c.requestBodys {
		results[i] = doRequestItem(body, httpClient)
		if c.trace != nil {
			c.trace(results[i])
		}
	}
	c.results <- results
}

func doRequestItem(body requestBody, httpClient http.Client) (result runResult) {
	var dnsStart, connectStart, reqStart time.Time
	var duration duration
	result = runResult{id: body.id}

	//now := time.Now()
	req, err := buildRequest(body)

	if err != nil {
		fmt.Println(err)
		result.err = runError{err.Error()}
		req.Close = true
	} else {
		trace := &httptrace.ClientTrace{
			DNSStart: func(info httptrace.DNSStartInfo) {
				dnsStart = time.Now()
			},
			DNSDone: func(info httptrace.DNSDoneInfo) {
				duration.dns = time.Now().Sub(dnsStart)
			},
			GetConn: func(hostPort string) {
				connectStart = time.Now()
			},
			GotConn: func(info httptrace.GotConnInfo) {
				duration.connect = time.Now().Sub(connectStart)
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				reqStart = time.Now()
			},
			GotFirstResponseByte: func() {
				duration.request = time.Now().Sub(reqStart)
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		res, err := httpClient.Do(req)
		if err == nil {
			result.duration = duration
			result.status = res.StatusCode
			result.statusMessage = res.Status
			result.headers = res.Header
			content, err := ioutil.ReadAll(res.Body)
			if err != nil {
				result.err = runError{err.Error()}
			} else {
				result.body = string(content)
			}
			defer res.Body.Close()
		}
	}
	return
}

func buildRequest(reqBody requestBody) (*http.Request, error) {
	var bodyReader io.Reader
	if reqBody.body != "" {
		bodyReader = strings.NewReader(reqBody.body)
	}
	req, err := http.NewRequest(reqBody.method, reqBody.url, bodyReader)
	if err != nil {
		return nil, err
	}
	headers := make(http.Header)
	for k, v := range reqBody.headers {
		headers.Set(k, v)
	}
	req.Header = headers
	return req, nil
}
