package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
)

func main3() {

	ln, err := net.Listen("tcp", ":50998")
	defer ln.Close()
	if err != nil {
		log.Fatal(err)
	}

	// loop and wait for connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error establishing a connection.")
		}
		data, err := ioutil.ReadAll(conn)
		if err != nil {
			fmt.Println("Read from TCP connection failed")
			fmt.Println(err)
			return
		}
		fmt.Println("From TCP:")
		fmt.Println(string(data))
	}
}
