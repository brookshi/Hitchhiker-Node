package hlog

import (
	"io"
	"log"
	"os"
)

var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

func init() {
	file, err := os.OpenFile("client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("open log file failed: ", err)
	}

	Info = log.New(io.MultiWriter(os.Stdout, file), "info: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(io.MultiWriter(os.Stdout, file), "warn: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(io.MultiWriter(os.Stderr, file), "error: ", log.Ldate|log.Ltime|log.Lshortfile)
}
