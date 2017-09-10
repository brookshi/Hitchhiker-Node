package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/brookshi/Hitchhiker-Node/hlog"
)

const (
	msg_hardware = iota
	msg_task
	msg_start
	msg_runResult
	msg_stop
)

const (
	status_idle = iota
	status_ready
	status_working
	status_finish
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
	status    byte
	code      byte
	testCase  testCase
	runResult runResult
	cpuNum    int
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
			c.send(message{status: status_idle, code: msg_hardware, cpuNum: runtime.NumCPU()})
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
	switch msg.code {
	case msg_task:
		c.testCase = msg.testCase
		c.testCase.trace = func(rst runResult) {
			hlog.Info.Println("trace")
			go c.send(message{status: status_working, code: msg_runResult, runResult: rst})
		}
		c.send(message{status: status_ready})
		hlog.Info.Println("status: ready")
	case msg_start:
		c.send(message{status: status_working})
		c.testCase.Run()
	case msg_stop:
		c.finish()
	}
}

func (c *client) send(msg message) {
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.BigEndian, msg)
	c.conn.Write(bin_buf.Bytes())
}

func (c *client) finish() {
	c.testCase.stop()
	c.send(message{status: status_finish})
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
