package main

import (
	"runtime"

	"github.com/brookshi/Hitchhiker-Node/hlog"
)

func main() {
	hlog.Init()
	runtime.GOMAXPROCS(runtime.NumCPU())
	hlog.Info.Println("set max procs: ", runtime.NumCPU())
	client := &client{}
	client.Do()
}
