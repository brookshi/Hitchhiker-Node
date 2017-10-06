package main

import (
	"runtime"
	"time"

	"github.com/brookshi/Hitchhiker-Node/hlog"
	"github.com/robertkrimen/otto"
)

func main() {
	hlog.Init()
	runtime.GOMAXPROCS(runtime.NumCPU())
	hlog.Info.Println("set max procs: ", runtime.NumCPU())
	// client := &client{}
	// client.Do()
	hlog.Info.Println("start")
	vm := otto.New()
	hlog.Info.Println("new vm")
	start := time.Now()
	v, err := vm.Run(`
			var responseCode = {code: 200};
			var responseObj = {success: false, result: {id: "001"}}
			var tests = {};
			tests["Status code is 200"] = responseCode.code === 200;
			tests["request is success"] = responseObj.success;
			tests["id is correct"] = responseObj.result.id === "001";
			result = JSON.stringify(tests);
		
		`)

	duration := time.Now().Sub(start)
	hlog.Info.Println(duration.Nanoseconds())
	hlog.Info.Println("end")
	hlog.Info.Println(v)
	hlog.Info.Println(err)
	hlog.Info.Println(vm.Get("result"))

	time.Sleep(time.Duration(1) * time.Hour)
}
