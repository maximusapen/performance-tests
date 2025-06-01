/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
)

var (
	terminate      bool
	fileLimit      int
	holdSeconds    int
	increaseRLimit bool
	fileFudge      int
)

func main() {

	// TODO Provide a mode by which files are opened after another process gives up a file descriptor
	flag.IntVar(&fileLimit, "files", 10, "total number files to open")
	flag.IntVar(&holdSeconds, "hold", 0, "Number of seconds to wait after all files are open")
	flag.BoolVar(&increaseRLimit, "rLimit", false, "Increase rLimit if greater than default rLimit")

	flag.Parse()

	fmt.Println("Run the following to find the maximum number of system file descriptors: cat /proc/sys/fs/file-max")

	// Go opens files before running any of this code
	fileFudge = 4

	if increaseRLimit {
		var rLimit syscall.Rlimit
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			fmt.Println("Error Getting Rlimit ", err)
		}
		fmt.Println("Current rLimit", rLimit)

		var required = uint64(fileLimit)
		if rLimit.Cur < required {
			if rLimit.Max < required {
				rLimit.Max = required
			}
			rLimit.Cur = required
			fmt.Println("Setting rLimit", rLimit)
			err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				fmt.Println("Error Setting Rlimit ", err)
				fmt.Println("You probably need to run as root")
			}
			err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				fmt.Println("Error Getting Rlimit ", err)
			}
			fmt.Println("Rlimit Final", rLimit)
		}
	}

	var fileBase = "./files/"
	var err error
	if _, err = os.Stat(fileBase); os.IsNotExist(err) {
		err = os.Mkdir(fileBase, 0750)
	}
	if err != nil {
		fmt.Printf("Couldn't create directory for for files: %s", err)
		os.Exit(1)
	}

	// Assume that fileFudge files has been opened as part of starting program
	var q = fileFudge
	for {
		c, err := os.Create(fileBase + strconv.Itoa(q))
		if err != nil {
			fmt.Println("All done at ", q)
			fmt.Println(err)
			break
		}
		defer c.Close()
		q++
		if q%int(fileLimit/10) == 0 {
			fmt.Println("Total open files: ", q)
		}
		if fileLimit > 0 && q >= fileLimit {
			fmt.Println("Limit reached")
			break
		}
	}
	if holdSeconds > 0 {
		fmt.Println("Hold time")
		time.Sleep(time.Duration(holdSeconds) * time.Second)
	}
	os.Exit(0)
}
