package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type testCase struct {
	body             requestBody
	totalCount       int
	concurrencyCount int
	qps              int
	timeout          int
	keepAlive        bool
	results          chan runResult
	finishChannel    chan struct{}
}

type requestBody struct {
	method  string
	url     string
	body    string
	headers map[string]string
	tests   string
}

type runResult struct {
	rid           string
	id            string
	envId         string
	err           runError
	body          string
	status        int
	statusMessage string
	elapsed       duration
	headers       map[string]string
	cookies       []string
	host          string
	tests         map[string]bool
	variables     map[string]interface{}
}

type duration struct {
	dns     time.Duration
	connect time.Duration
	request time.Duration
}

type runError struct {
	message string
}

// Request is a http request
func (c *testCase) doRequest() {
	now := time.Now()
	var dnsStart, connectStart time.Time
	var duration duration

	req, err := buildRequest(c.body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	req.Close = true

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	defer res.Body.Close()

	content, _ := ioutil.ReadAll(res.Body)
	return string(content), nil
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
