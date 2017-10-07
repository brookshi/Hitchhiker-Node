package main

import (
	"runtime"
	"strings"
	"time"

	"github.com/brookshi/Hitchhiker-Node/hlog"
)

func main() {
	hlog.Init()
	runtime.GOMAXPROCS(runtime.NumCPU())
	hlog.Info.Println("set max procs: ", runtime.NumCPU())
	// client := &client{}
	// client.Do()
	hlog.Info.Println("start")
	//vm := otto.New()
	//ctx := duktape.New()
	hlog.Info.Println("new vm")
	start := time.Now()
	//aa := "1234"
	// ctx.PushGlobalGoFunction("getAA", func(c *duktape.Context) int {
	// 	c.PushString(aa)
	// 	return 0
	// })
	//ctx.PushString(aa)
	//ctx.PutGlobalString("aaa")
	//duk_eval_string(ctx, "(function (aa) { Duktape.aa = aa; })")

	//err := ctx.PevalString(` result = aaa;`)

	runResult := runResult{
		Body:          `{"success": true, "result": { "id": "001"}}`,
		Status:        200,
		StatusMessage: "normal",
		Duration:      duration{Connect: 1000000, DNS: 2000000, Request: 30000000},
		Headers:       map[string]string{"title": strings.Join([]string{"111"}, ";")},
	}

	hlog.Info.Println("new runResult")

	result := ottoInterpret(`
		tests["Status code is 200"] = responseCode.code === 200;
		tests["request is success"] = responseObj.success;
		tests["id is correct"] = responseObj.result.id === "001";
		tests["header is right"] = responseHeaders.title === "1112";
		tests["status is 200"] = responseCode.code === 200;
		tests["msg is normal"] = responseCode.name === 'normal'; 
		$variables$.hbsm_a = responseCode.name;
	`, runResult)
	// v, err := vm.Run(`
	// 		var responseCode = {code: 200};
	// 		var responseObj = {success: false, result: {id: "001"}}
	// 		var tests = {};
	// 		tests["Status code is 200"] = responseCode.code === 200;
	// 		tests["request is success"] = responseObj.success;
	// 		tests["id is correct"] = responseObj.result.id === "001";
	// 		result = JSON.stringify(tests);

	// 	`)
	//result := ctx.GetString(-1)
	duration := time.Now().Sub(start)
	hlog.Info.Println(duration.Nanoseconds())
	hlog.Info.Println("end")
	hlog.Info.Println(result)
	//hlog.Info.Println(err)
	// hlog.Info.Println(vm.Get("result"))

	time.Sleep(time.Duration(1) * time.Hour)
}
