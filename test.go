package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/tarm/serial"
)

/**
 * Starts a goroutine server which will sit and wait for a connection
 * then respond to the request as necessary
 * @param port Serial port to listen on
 */
func send(port *serial.Port) {

	n, _ := port.Write([]byte("\x00\xFA\x00\x00\x00\x0b\x08serial test"))
	fmt.Printf("Sent %d bytes\n", n)

	// we'll sit here and wait for something to come along our port
	metabuf := make([]byte, 7)
	n, err := port.Read(metabuf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(metabuf)

	// look up reported message size
	msgsize := binary.BigEndian.Uint32(metabuf[2:6])
	fmt.Println(msgsize)
	msgbuf := make([]byte, msgsize)

	// read from buffer
	n, err = port.Read(msgbuf)
	message := string(msgbuf)
	fmt.Println("message recieved: " + message)

	//c, _ := net.Dial("tcp", ":50999")
	//m, _ := c.Write([]byte("\x00\xFA\x00\x00\x00\x08\x00tcp test"))
	//fmt.Printf("Sent %d bytes\n", m)
	//c.Close()
}

func recv() {

}

func main2() {

	// access the applications arguments from startup
	args := os.Args[1:]

	// set default port to try
	name := "COM4"

	// get name of attempted COM port
	if len(args) > 0 {
		name = args[1]
	}

	// create the config object
	config := &serial.Config{Name: name, Baud: 9600}
	// open the port
	port, err := serial.OpenPort(config)

	if err != nil {
		log.Fatal(err)
	}

	// start the server go routine
	send(port)

	recv()
}
