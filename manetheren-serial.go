package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/tarm/serial"
)

// create an enumeration to check which type of service request is received
const (
	// if rx, process response
	// if tx, send response
	serveWeather  uint8 = iota // 0
	serveForecast uint8 = iota // 1
	serveQuote    uint8 = iota // 2
	serveTime     uint8 = iota // 3
	serveCalendar uint8 = iota // 4
	serveTasks    uint8 = iota // 5
	serveConfig   uint8 = iota // 6
	serveOther    uint8 = iota // 7

	// if rx, process request
	// if tx, making request
	requestWeather  uint8 = iota // 8
	requestForecast uint8 = iota // 9
	requestQuote    uint8 = iota // 10
	requestTime     uint8 = iota // 11
	requestCalendar uint8 = iota // 12
	requestTasks    uint8 = iota // 13
	requestConfig   uint8 = iota // 14
	requestOther    uint8 = iota // 15
)

// start byte is 0
const startByte byte = 0x00

// identify that we're using manetheren services
const manetherenByte byte = 0xFA

// end byte 255
const endByte byte = 0xFF

// header length
const headerLength int = 7

// port to use for tcp server comm
const tcpServerPort string = "50999"

// port to send responses to a client
const tcpClientPort string = "50998"

/*
 * manetheren-serial packet structure
 * STARTBYTE | MANETHERENBYTE | LENGTH 4xbyte | TYPE 1xbyte | MSG
 * 1 + 1 + 4 + 1 + # =    7 + # bytes
 */

/**
 * Starts a goroutine server which will sit and wait for a connection
 * then respond to the request as necessary
 * @param port Serial port to listen on
 * @param wg WaitGroup to check for go routine completion
 */
func serialServer(port *serial.Port, wg sync.WaitGroup) {

	defer wg.Done()

	// sit and wait for inputs
	var prevByte byte = 0x00
	var nextBuf = make([]byte, 1)
	// if it is a good start sequence, lets read it
	var metabuf = make([]byte, headerLength-2)

	for {

		// read single lines in until we find our starting point
		_, err := port.Read(nextBuf)
		if err != nil {
			log.Fatal(err)
		}

		// if it isn't the start of our sequence move along
		if nextBuf[0] != manetherenByte || prevByte != startByte {
			prevByte = nextBuf[0]
			continue
		}

		// we're going to scan until we find the right format

		n, err := io.ReadFull(port, metabuf)
		if err != nil {
			log.Fatal(err)
		}

		// look up reported message size
		msgsize := binary.BigEndian.Uint32(metabuf[:4])
		fmt.Println(msgsize)
		msgbuf := make([]byte, msgsize)

		// read from buffer
		n, err = io.ReadFull(port, msgbuf)
		if err != nil {
			fmt.Println(err)
			msgbuf = nil
			continue
		}
		message := string(msgbuf)
		msgbuf = nil

		fmt.Printf("Bytes read: %d, message recieved %s\n", n, message)
		go handleSerialMessage(port, uint8(metabuf[4]), message)
		// handleSerialMessage(port, uint8(metabuf[4]), message)
	}
}

/**
 * Handle an incoming serial message
 * @param port port to send reply over
 * @param msgtype type of message sent over the wire
 * @param message message received for processing
 */
func handleSerialMessage(port *serial.Port, msgtype uint8, message string) {

	// base request message
	if msgtype >= requestWeather {
		// get the requested service
		response := manetherenResponse(msgtype, message)
		// calculate the response service type
		serveType := msgtype - requestWeather
		// send back as a service type
		serialSend(port, serveType, response)
	} else {
		// handle a serial response from the server with the data
		// we need to pass along
		// the display should open a
		tcpSend(msgtype, message)
	}

}

/**
 * Send a message across the port we've openned for the other side
 * to processes
 * @param port The serial port to transmit data through
 * @param msgtype our constant message type to process
 * @param message the data to transmit across the wire
 */
func serialSend(port *serial.Port, msgtype uint8, message string) {

	var strlen uint32 = uint32(len(message))
	sendbuf := make([]byte, int(strlen)+headerLength)
	sizebuf := make([]byte, 4)

	// put the start bit
	sendbuf[0] = startByte
	sendbuf[1] = manetherenByte

	binary.BigEndian.PutUint32(sizebuf, strlen)
	copy(sendbuf[2:6], sizebuf)

	sendbuf[6] = byte(msgtype)
	copy(sendbuf[7:int(strlen)+headerLength], []byte(message))

	n, err := port.Write(sendbuf)
	if err != nil {
		log.Println("failed to send")
		log.Println(err)
	}

	//fmt.Print(sendbuf)
	fmt.Printf("Bytes sent: %d\n", n)
}

/**
 * Using local TCP ports to conect to display, so just send to
 * clilent port over local using a JSON format
 * @param msgType constant message type to interpret
 * @param message the message to transmit
 */
func tcpSend(msgType uint8, message string) {

	// open the TCP connection
	conn, err := net.Dial("tcp", ":"+tcpClientPort)
	defer conn.Close()
	if err != nil {
		fmt.Println("TCP connection to local display failed")
		return
	}

	// send a JSON message with served data
	fmt.Fprintf(conn, "{type:\"%s\",message:%s}",
		servicePath(msgType, ""),
		message)
}

/**
 * Handle an incoming serial message
 * @param port port to send reply over
 * @param msgtype type of message sent over the wire
 * @param message message received for processing
 */
func handleTCPMessage(port *serial.Port, msgtype uint8, message string) {
	// if we are the far end client we'll be making requests
	// the request message over TCP comes from the display
	// requesting an update from the manetheren server
	// so we will just forward that request over our serial interface
	// to the far end
	if msgtype >= requestWeather {
		serialSend(port, msgtype, message)
		fmt.Println("TCP Request complete")
	} else {
		// but if we are on the manetheren end, we still want
		// this servelet interface to publish a forced update
		// so if the message type is serve over TCP
		// pretend to receive a serial message
		reqType := msgtype + requestWeather
		handleSerialMessage(port, reqType, message)
	}
}

/**
 * Interprocess communication handled using a socket
 * @param port Still need to know what port to forward data to
 * @param wg WaitGroup to wait for go routine completion
 */
func tcpServer(port *serial.Port, wg sync.WaitGroup) {
	defer wg.Done()

	ln, err := net.Listen("tcp", ":"+tcpServerPort)
	defer ln.Close()
	if err != nil {
		log.Fatal(err)
	}

	// loop and wait for connections
	for {
		fmt.Println("Listening for new connection")
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error establishing a connection.")
		}

		fmt.Println("Connection received")
		go connectionRequest(port, conn)
		// connectionRequest(port, conn)
	}
}

/**
 * Handle an incoming connection request and process send/recv data
 * @param conn The network connection to receive request from
 */
func connectionRequest(port *serial.Port, conn net.Conn) {
	defer conn.Close()
	data, err := ioutil.ReadAll(conn)
	if err != nil {
		fmt.Println("Read from TCP connection failed")
		fmt.Println(err)
		return
	}
	fmt.Println("From TCP:")
	fmt.Println(string(data))

	// clear the buffer if our format isn't good
	if len(data) < headerLength ||
		data[0] != startByte ||
		data[1] != manetherenByte {
		fmt.Println("Improper message format over TCP")
		return
	}

	// look up reported message size
	msgsize := binary.BigEndian.Uint32(data[2:6])
	fmt.Printf("TCP Message size: %d\n", msgsize)

	// we'll want to block I think
	handleTCPMessage(port, uint8(data[6]), string(data[7:]))
	fmt.Println("connection request complete")
}

// returns the service type string, based on the msgtype
func servicePath(msgtype uint8, extension string, otherService ...string) string {

	// adjust the type to match either serve or request
	adjustedType := msgtype
	if msgtype < requestWeather {
		adjustedType = msgtype + requestWeather
	}

	extra := ""
	if len(otherService) > 0 {
		extra = otherService[0]
	}

	// check for possibilities
	switch adjustedType {
	case requestWeather:
		return "weather" + extension + extra
	case requestForecast:
		return "forecast" + extension + extra
	case requestQuote:
		return "quote" + extension + extra
	case requestTime:
		return "time" + extension + extra
	case requestCalendar:
		return "calendar" + extension + extra
	case requestTasks:
		return "tasks" + extension + extra
	case requestConfig:
		if len(otherService) == 0 {
			fmt.Println("Not enough values to other service request")
			return ""
		}
		return "config" + extension + extra
	case requestOther:
		if len(otherService) == 0 {
			fmt.Println("Not enough values to other service request")
			return ""
		}
		return extra
	default:
		fmt.Println("Improper message type")
		return ""
	}
}

/**
 * Make a request of the manetheren server running on localhost
 * @param msgtype the type of response we are requesting
 * @param otherService if some so far undefined service is needed
 * @return string response from manetheren cache/proxy, usually JSON
 */
func manetherenResponse(msgtype uint8, otherService ...string) string {
	// build the path based on service type
	var other string = ""
	if len(otherService) > 0 {
		other = otherService[0]
	}

	sPath := servicePath(msgtype, ".php", other)
	if sPath == "" {
		return ""
	}
	var path string = "http://manetheren/services/" + sPath

	// make HTTP request
	resp, err := http.Get(path)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	// read the response from the manetheren proxy
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body)
}

func main() {

	// access the applications arguments from startup
	args := os.Args[1:]

	// set default port to try
	name := "COM3"

	// get name of attempted COM port
	if len(args) > 0 {
		name = args[0]
	}

	// create the config object
	config := &serial.Config{Name: name, Baud: 9600}
	// open the port
	port, err := serial.OpenPort(config)
	defer port.Close()

	if err != nil {
		log.Fatal(err)
	}

	// we'll wait for server to finish its business
	var wg sync.WaitGroup

	// start the server go routine
	wg.Add(1)
	go serialServer(port, wg)

	// start TCP server for internal comms
	wg.Add(1)
	go tcpServer(port, wg)

	// wait for completion of go routines
	wg.Wait()
}
