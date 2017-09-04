package main

import (
	"fmt"
	"time"
	//"github.com/ddliu/go-httpclient"
)

func main() {
	start := time.Now()
	ch := make(chan string)
	contents := make([]string, 10)
	for i := 0; i < 10; i++ {
		go test(ch)

	}
	sec := time.Since(start).Seconds()
	for i := 0; i < 10; i++ {
		contents[i] = <-ch
	}
	fmt.Printf("%.4fs elapsed \n", sec)
	sec = time.Since(start).Seconds()
	fmt.Printf("%.4fs total elapsed \n", sec)
	fmt.Printf("length: %d \n", len(contents))
	fmt.Printf("first content: %s", contents[0])
	fmt.Printf("last content: %s", contents[9])
}

func test(ch chan<- string) {
	content, err := Request("GET", "https://httpbin.org/get", "")
	if err != nil {
		fmt.Println(err)
		ch <- ""
		return
	}
	//content, err := res.ToString()
	if err != nil {
		fmt.Println(err)
		ch <- ""
		return
	}
	ch <- content
}
