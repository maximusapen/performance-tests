/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package controller

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

// CommandType defines the comand control bytes
type CommandType byte

// START, STOP and TERMINATE metrics gathering control bytes
const (
	START     CommandType = 0x1
	STOP                  = 0x2
	TERMINATE             = 0x3
)

func (cmd CommandType) String() string {
	switch cmd {
	case START:
		return "START"
	case STOP:
		return "STOP"
	case TERMINATE:
		return "TERMINATE"
	}
	return fmt.Sprintf("INVALID: %x", byte(cmd))
}

func (cmd CommandType) valid() bool {
	return !strings.HasPrefix(fmt.Sprintf("%s", cmd), "INVALID")
}

// Communications channel for controller
var controlChan chan CommandType

func check(e error) {
	if e != nil {
		if e == io.EOF {
			// Tolerate EOF
			return
		}
		log.Fatalln(e.Error())
	}
}

// handleCommand initiates processing of received control commands
func handleCommand(c net.Conn) bool {
	defer c.Close()

	// Make a buffer to hold incoming data.
	buf := make([]byte, 1)

	// Read the incoming connection into the buffer.
	bytesReceived, err := c.Read(buf)
	check(err)

	terminate := false
	if bytesReceived == 1 {
		controlVal := CommandType(buf[0])
		log.Printf("Cruiser Metrics : %s\n", controlVal)

		// Controller termination request ?
		terminate = (controlVal == TERMINATE)

		// If valid, send the received control command back
		if controlVal.valid() {
			controlChan <- controlVal
		}
	}
	return terminate
}

// Start is the entrypoint for the cruiser metrics controller
// It listens for user control commands and invokes the processing logic
func Start(controlPort int, control chan CommandType) {
	controlChan = control

	log.Printf("Cruiser Metrics : Listening for control commands on port %d\n", controlPort)
	ln, err := net.Listen("tcp", strings.Join([]string{"localhost", strconv.Itoa(controlPort)}, ":"))
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()

	terminate := false
	for !terminate {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		terminate = handleCommand(conn)
	}
}
