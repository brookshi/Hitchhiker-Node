package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type testCase struct {
	RequestBodyList  []requestBody `json:"requestBodyList"`
	TotalCount       int           `json:"totalCount"`
	ConcurrencyCount int           `json:"concurrencyCount"`
	QPS              int           `json:"qps"`
	Timeout          int           `json:"timeout"`
	results          chan []runResult
	forceStop        int32
	trace            func(rst runResult)
}

type requestBody struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Tests   string            `json:"tests"`
}

type runResult struct {
	ID            string          `json:"id"`
	Err           runError        `json:"error"`
	Body          string          `json:"body"`
	Status        int             `json:"status"`
	StatusMessage string          `json:"statusMessage"`
	Duration      duration        `json:"duration"`
	Headers       http.Header     `json:"headers"`
	Tests         map[string]bool `json:"tests"`
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
	c.results = make(chan []runResult, c.TotalCount)
	atomic.StoreInt32(&c.forceStop, 0)
	c.start()
	close(c.results)
}

func (c *testCase) start() {
	var waiter sync.WaitGroup
	waiter.Add(c.ConcurrencyCount)

	for i := 0; i < c.ConcurrencyCount; i++ {
		go func() {
			c.work(c.TotalCount / c.ConcurrencyCount)
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

	httpClient := http.Client{Timeout: time.Duration(c.Timeout) * time.Second}
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
	for i, body := range c.RequestBodyList {
		results[i] = doRequestItem(body, httpClient)
		if c.trace != nil {
			c.trace(results[i])
		}
	}
	c.results <- results
}

func (c *testCase) stop() {
	atomic.CompareAndSwapInt32(&c.forceStop, 0, 1)
}

func doRequestItem(body requestBody, httpClient http.Client) (result runResult) {
	var dnsStart, connectStart, reqStart time.Time
	var duration duration
	result = runResult{ID: body.ID}

	//now := time.Now()
	req, err := buildRequest(body)

	if err != nil {
		fmt.Println(err)
		result.Err = runError{err.Error()}
		req.Close = true
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
			result.StatusMessage = res.Status
			result.Headers = res.Header
			content, err := ioutil.ReadAll(res.Body)
			if err != nil {
				result.Err = runError{err.Error()}
			} else {
				result.Body = string(content)
			}
			defer res.Body.Close()
		}
	}
	return
}

func buildRequest(reqBody requestBody) (*http.Request, error) {
	var bodyReader io.Reader
	if reqBody.Body != "" {
		bodyReader = strings.NewReader(reqBody.Body)
	}
	req, err := http.NewRequest(reqBody.Method, reqBody.URL, bodyReader)
	if err != nil {
		return nil, err
	}
	headers := make(http.Header)
	for k, v := range reqBody.Headers {
		headers.Set(k, v)
	}
	req.Header = headers
	return req, nil
}
