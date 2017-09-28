package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/brookshi/Hitchhiker-Node/hlog"
)

const (
	msgHardware = iota
	msgTask
	msgStart
	msgRunResult
	msgStop
	msgStatus
)

const (
	statusIdle = iota
	statusReady
	statusWorking
	statusFinish
	statusDown
)

type config struct {
	Address  string
	Interval time.Duration
}

type client struct {
	conn     net.Conn
	errChan  chan bool
	testCase testCase
}

type message struct {
	Status    byte      `json:"status"`
	Code      byte      `json:"code"`
	TestCase  testCase  `json:"testCase"`
	RunResult runResult `json:"runResult"`
	CPUNum    int       `json:"cpuNum"`
}

func (c *client) Do() {
	hlog.Info.Println("read config")
	config, err := readConfig()
	if err != nil {
		hlog.Error.Println("read config file:", err)
		os.Exit(1)
	}

	for {
		c.errChan = make(chan bool)
		throttle := time.Tick(config.Interval * time.Second)
		c.conn, err = net.Dial("tcp", config.Address)
		if err != nil {
			hlog.Error.Println("connect:", err)
			go func() { c.errChan <- true }()
		} else {
			hlog.Info.Println("connect: success")
			c.send(message{Status: statusIdle, Code: msgHardware, RunResult: runResult{ID: "1"}, CPUNum: runtime.NumCPU()})
			hlog.Info.Println("status: idle")
			go c.read()
		}
		<-c.errChan
		<-throttle
		hlog.Info.Println("retry")
	}
}

func (c *client) read() {
	defer c.conn.Close()
	for {
		reader := bufio.NewReader(c.conn)
		content, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			hlog.Error.Println("read:", err)
			c.testCase.stop()
			c.errChan <- true
			return
		}
		hlog.Info.Println("read: ", string(content))
		var msg message
		json.Unmarshal(content, &msg)
		go c.handleMsg(msg)
	}
}

func (c *client) handleMsg(msg message) {
	switch msg.Code {
	case msgTask:
		c.testCase = msg.TestCase
		c.testCase.trace = func(rst runResult) {
			hlog.Info.Println("trace")
			go c.send(message{Status: statusWorking, Code: msgRunResult, RunResult: rst})
		}
		c.send(message{Status: statusReady, Code: msgStatus})
		hlog.Info.Println("status: ready")
	case msgStart:
		c.send(message{Status: statusWorking, Code: msgStatus})
		c.testCase.Run()
		c.finish()
	case msgStop:
		c.finish()
	}
}

func (c *client) send(msg message) {
	buf, err := json.Marshal(msg)
	if err != nil {
		hlog.Error.Println("stringify message error: ", err)
		return
	}
	hlog.Info.Println("send message: ", string(buf))
	c.conn.Write(buf)
}

func (c *client) finish() {
	c.testCase.stop()
	c.send(message{Status: statusFinish, Code: msgStatus})
}

func readConfig() (config, error) {
	var c config
	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		hlog.Error.Println("read config failed: ", err)
		return c, err
	}
	err = json.Unmarshal(file, &c)
	return c, err
}
