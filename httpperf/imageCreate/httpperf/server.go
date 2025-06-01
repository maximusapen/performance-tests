
package main

// A GO HTTP server for httpperf. Listens on port 8080 and 8443 for /hello and /request.
// /hello returns "Hello World and /request takes a single query parameter, 'size' that specifies
// the size of the response that should be returned. Both URLs accept arbitrarily large
// response bodies which can be used for uploads.
import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var ops uint64
var hostname string

const isTrue = "true"

func hello(w http.ResponseWriter, r *http.Request) {
	debug := r.URL.Query()["debug"]
	var st time.Time
	if debug != nil {
		if debug[0] == isTrue {
			st = time.Now()
		}
	}
	defer r.Body.Close()
	//fmt.Println(r.URL.String())

	// discard request data, if any
	_, _ = io.Copy(ioutil.Discard, r.Body)

	io.WriteString(w, "Hello world!")
	if debug != nil {
		if debug[0] == isTrue {
			count := atomic.AddUint64(&ops, 1)
			respTime := time.Since(st).Nanoseconds() / 1000
			log.Printf("Hello world! host: %v , count: %v , time: %v \n", hostname, count, respTime)
		}
	}
}

func dumpStacks(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Println("Dumping stacks for all goroutines")
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, true)
	log.Printf("%s", buf)
	_, _ = io.Copy(ioutil.Discard, r.Body)
	io.WriteString(w, string(buf[:]))
}

func request(w http.ResponseWriter, r *http.Request) {
	debug := r.URL.Query()["debug"]
	var st time.Time
	if debug != nil {
		if debug[0] == "true" {
			st = time.Now()
		}
	}
	defer r.Body.Close()
	//fmt.Println(r.URL.String())

	// discard request data, if any
	_, _ = io.Copy(ioutil.Discard, r.Body)

	var size = 1
	tmp := r.URL.Query()["size"]

	if tmp != nil {
		if s, err := strconv.Atoi(tmp[0]); err == nil {
			size = s
		}
	}

	if size < 1 {
		size = 1
	}

	fmt.Fprintln(w, size)
	response := randomBytes(size)

	w.Write(response)
	if debug != nil {
		if debug[0] == "true" {
			count := atomic.AddUint64(&ops, 1)
			respTime := time.Since(st).Nanoseconds() / 1000
			log.Printf("Host: %v , count: %v , time: %v, content: %v\n", hostname, count, respTime, string(response[:]))
		}
	}
}

func main() {
	httpPort := flag.String("httpPort", "8080", "http port")
	httpsPort := flag.String("httpsPort", "8443", "https port")
	flag.Parse()
	hostname, _ = os.Hostname()
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/request", request)
	http.HandleFunc("/dumpStacks", dumpStacks)
	go http.ListenAndServe(":"+*httpPort, nil)
	log.Fatal(http.ListenAndServeTLS(":"+*httpsPort, "server.pem", "server.key", nil))
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const letterCount = len(letterBytes)

var mutex sync.RWMutex
var randomBySize = make(map[int][]byte)

func randomBytes(n int) []byte {
	mutex.RLock()
	b, exists := randomBySize[n]
	mutex.RUnlock()

	if !exists {
		//fmt.Println("creating new", n, "sized array")
		b = make([]byte, n)

		for i := range b {
			b[i] = letterBytes[rand.Intn(letterCount)]
		}

		mutex.Lock()
		// no need to check for race condition of 2 readers getting !exists
		// just duplicate and replace
		randomBySize[n] = b
		mutex.Unlock()
	}

	return b
}
