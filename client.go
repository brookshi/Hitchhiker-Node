package main

import (
	"encoding/json"
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
	msgFileStart
	msgFileFinish
)

const (
	statusIdle = iota
	statusReady
	statusWorking
	statusFinish
	statusDown
	statusFileReady
)

const dataFilePath = "./global_data.zip"

type config struct {
	Address  string
	Interval time.Duration
}

type client struct {
	conn     *websocket.Conn
	errChan  chan bool
	testCase testCase
	isFile   bool
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
		if c.isFile {
			c.receiveFile()
		} else {
			c.receiveJSON()
		}
	}
}

func (c *client) receiveFile() {
	var data []byte
	err := websocket.Message.Receive(c.conn, &data)
	if err != nil {
		c.readErr(err)
		return
	}
	if len(data) == 3 && data[0] == 36 {
		c.isFile = false
		unzip(dataFilePath, "global_data/")
		go c.handleMsg(message{Type: msgFileFinish})
	} else {
		err = saveFile(data)
		if err != nil {
			c.readErr(err)
			return
		}
	}
}

func (c *client) receiveJSON() {
	var msg message
	err := websocket.JSON.Receive(c.conn, &msg)
	if err != nil {
		c.readErr(err)
		return
	}
	buf, _ := json.Marshal(msg)
	hlog.Info.Println("read: ", string(buf))
	if msg.Type == msgFileStart {
		hlog.Info.Println("status: file start")
		c.isFile = true
	}
	go c.handleMsg(msg)
}

func (c *client) readErr(e error) {
	hlog.Error.Println("read:", e)
	c.testCase.stop()
	c.errChan <- true
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
	case msgFileFinish:
		hlog.Info.Println("status: file finish")
		c.send(message{Status: statusFileReady, Type: msgStatus})
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
	hlog.Info.Println("send request run result", msg.Type)
	c.conn.Write(buf)
}

func (c *client) finish() {
	c.testCase.stop()
	c.send(message{Status: statusFinish, Type: msgStatus})
}
