/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Scan a nmon file and report in unexpected gaps in collecting data

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	nmonFile := os.Stdin

	if len(os.Args) > 1 && len(os.Args[1]) > 0 {
		nmonFilePath := os.Args[1]

		// Open files
		var err error
		// #nosec G304
		nmonFile, err = os.Open(nmonFilePath)
		if err != nil {
			fmt.Println("ERROR: Opening file: ", err)
			os.Exit(1)
		}
	}

	var standardDelta time.Duration
	var lastLine string
	var lastTime time.Time

	scanner := bufio.NewScanner(nmonFile)

	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "AAA,interval") {
			items := strings.Split(scanner.Text(), ",")
			seconds, err := strconv.ParseInt(items[2], 10, 64)
			if err != nil {
				fmt.Printf("Bad conversion of interval: %s - %v\n", items[2], err)
			}
			standardDelta = time.Duration(seconds) * time.Second
			//fmt.Printf("Expected interval is %d seconds\n", seconds)
			break
		}
	}

	minReportableDelta := standardDelta + 2*time.Second

	for scanner.Scan() {
		// Find each of these lines: ZZZZ,T0001,16:20:52,05-DEC-2017
		if strings.HasPrefix(scanner.Text(), "ZZZZ,T") {
			items := strings.Split(scanner.Text(), ",")

			// Extract the time from the line
			currentTime, err := time.Parse("15:04:05,02-Jan-2006", items[2]+","+items[3])
			if err != nil {
				fmt.Printf("ERROR: Bad conversion of time %s - %v\n", items[2]+","+items[3], err)
			}
			//fmt.Println(currentTime)

			// If no previous delta then just save time and continue
			if len(lastLine) != 0 {
				// Calculate delta and compare to expected delta, if changed print this and previos T#, Time, Delta
				currentDelta := currentTime.Sub(lastTime)

				if currentDelta != standardDelta && currentDelta >= minReportableDelta && items[1] != "T0003" {
					fmt.Printf("%s %s %s %v\n", items[1], items[3], items[2], currentDelta-standardDelta)
				}
			}

			lastTime = currentTime
			lastLine = scanner.Text()
		}
	}
}
