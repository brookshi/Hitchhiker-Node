package main

import (
	"encoding/json"
	"fmt"
	"time"

	"fknsrs.biz/p/ottoext/loop"
	"github.com/brookshi/Hitchhiker-Node/hlog"

	"github.com/robertkrimen/otto"
	// duktape "gopkg.in/olebedev/go-duktape.v2"
)

type testResult struct {
	Tests     map[string]bool   `json:"tests"`
	Variables map[string]string `json:"variables"`
}

const (
	Duktape = iota
	Otto
)

const template = `
	var responseObj = {};
	try {
		responseObj = JSON.parse(responseBody);
	} catch (e) {
		responseObj = e;
	}
	var responseCode = { code: responseCode_Status, name: responseCode_Msg };
	var responseHeaders = {};
	try {
		responseHeaders = JSON.parse(responseHeaders_Str);
	} catch (e) {
		responseHeaders = e;
	}

	var $variables$ = {};
	var tests = {};
	var $variables$ = {};
	var $export$ = function(obj){ };
	%s
	$$out = JSON.stringify({tests: tests, variables: $variables$});
`

func interpret(engine byte, jsStr string, runResult runResult) testResult {
	if engine == Otto {
		return ottoInterpret(jsStr, runResult)
	}

	return duktapeInterpret(jsStr, runResult)
}

func ottoInterpret(jsStr string, runResult runResult) (result testResult) {
	vm := otto.New()
	l := loop.New(vm)
	vm.Set("responseBody", runResult.Body)
	vm.Set("responseCode_Status", runResult.Status)
	vm.Set("responseCode_Msg", runResult.StatusMessage)
	headers, _ := json.Marshal(runResult.Headers)
	vm.Set("responseHeaders_Str", string(headers))
	vm.Set("responseTime", float64((runResult.Duration.Connect+runResult.Duration.DNS+runResult.Duration.Request).Nanoseconds())/float64(time.Millisecond))
	err := l.EvalAndRun(fmt.Sprintf(template, jsStr))
	testRst, err := vm.Get("$$out")
	result = testResult{
		Tests:     make(map[string]bool),
		Variables: make(map[string]string),
	}
	if err != nil {
		result.Tests[err.Error()] = false
	} else {
		json.Unmarshal([]byte(testRst.String()), &result)
	}
	for k, v := range result.Tests {
		hlog.Info.Println(k, v)
	}
	for k, v := range result.Variables {
		hlog.Info.Println(k, v)
	}
	return
}

func duktapeInterpret(jsStr string, runResult runResult) (result testResult) {
	// ctx := duktape.New()
	// ctx.PushString(runResult.Body)
	// ctx.PutGlobalString("responseBody")
	// ctx.PushNumber(float64((runResult.Duration.Connect + runResult.Duration.DNS + runResult.Duration.Request).Nanoseconds()) / float64(time.Millisecond))
	// ctx.PutGlobalString("responseTime")
	// ctx.PushNumber(float64(runResult.Status))
	// ctx.PutGlobalString("responseCode_Status")
	// ctx.PushString(runResult.StatusMessage)
	// ctx.PutGlobalString("responseCode_Msg")
	// headers, _ := json.Marshal(runResult.Headers)
	// ctx.PushString(string(headers))
	// ctx.PutGlobalString("responseHeaders_Str")

	// err := ctx.PevalString(fmt.Sprintf(template, jsStr))
	// data := ctx.GetString(-1)
	// if err != nil {
	// 	result.Tests[err.Error()] = false
	// } else {
	// 	json.Unmarshal([]byte(data), &result)
	// }
	return
}
