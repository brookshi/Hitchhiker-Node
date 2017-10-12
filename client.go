package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/brookshi/Hitchhiker-Node/hlog"
	"golang.org/x/net/websocket"
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
	conn     *websocket.Conn
	errChan  chan bool
	testCase testCase
}

type message struct {
	Status    byte      `json:"status"`
	Type      byte      `json:"type"`
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
		c.conn, err = websocket.Dial("ws://"+config.Address, "", "http://"+config.Address)
		if err != nil {
			hlog.Error.Println("connect:", err)
			go func() { c.errChan <- true }()
		} else {
			hlog.Info.Println("connect: success")
			c.send(message{Status: statusIdle, Type: msgHardware, RunResult: runResult{ID: "1"}, CPUNum: runtime.NumCPU()})
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
		var msg message
		err := websocket.JSON.Receive(c.conn, &msg)
		if err != nil {
			hlog.Error.Println("read:", err)
			c.testCase.stop()
			c.errChan <- true
			return
		}
		buf, _ := json.Marshal(msg)
		hlog.Info.Println("read: ", string(buf))
		go c.handleMsg(msg)
	}
}

func (c *client) handleMsg(msg message) {
	switch msg.Type {
	case msgTask:
		c.testCase = msg.TestCase
		c.testCase.trace = func(rst runResult) {
			hlog.Info.Println("trace")
			c.send(message{Status: statusWorking, Type: msgRunResult, RunResult: rst})
		}
		c.send(message{Status: statusReady, Type: msgStatus})
		hlog.Info.Println("status: ready")
	case msgStart:
		hlog.Info.Println("status: start")
		c.send(message{Status: statusWorking, Type: msgStatus})
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
	hlog.Info.Println("send request run result")
	c.conn.Write(buf)
}

func (c *client) finish() {
	c.testCase.stop()
	c.send(message{Status: statusFinish, Type: msgStatus})
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
