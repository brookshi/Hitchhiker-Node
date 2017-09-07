package main

import (
	"bufio"
	"encoding/json"
	"net"
	"os"

	"github.com/brookshi/Hitchhiker-Node/hlog"
)

const (
	msg_start = iota
	msg_task
)

type config struct {
	address string
}

type client struct {
}

type message struct {
	code byte
}

func (c *client) init() {
	config, err := readConfig()
	if err != nil {
		hlog.Error.Println("read config file:", err)
		os.Exit(1)
	}
	conn, err := net.Dial("tcp", config.address)
	if err != nil {
		hlog.Error.Println("conn:", err)
		os.Exit(1)
	}
	go c.read(conn)
}

func (c *client) read(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		content, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			hlog.Error.Println("read:", err)
			return
		}
		var msg message
		json.Unmarshal(content, &msg)
		c.handleMsg(msg)
	}
}

func (c *client) handleMsg(msg message) {

}

func readConfig() (*config, error) {
	c := &config{}
	file, err := os.Open("config.json")
	defer file.Close()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	return c, err
}
