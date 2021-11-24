  // Copyright (c) 2017 Aidos Developer

  // Permission is hereby granted, free of charge, to any person obtaining a copy
  // of this software and associated documentation files (the "Software"), to deal
  // in the Software without restriction, including without limitation the rights
  // to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
  // copies of the Software, and to permit persons to whom the Software is
  // furnished to do so, subject to the following conditions:

  // The above copyright notice and this permission notice shall be included in
  // all copies or substantial portions of the Software.

  // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
  // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
  // FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
  // AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
  // LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
  // OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
  // THE SOFTWARE.

  package main

  import (
  	"bytes"
  	"flag"
  	"fmt"
  	"github.com/AidosKuneen/adk-daemon/aidos"
  	"github.com/gorilla/rpc"
  	"github.com/gorilla/rpc/json"
  	"log"
  	"net/http"
  	_ "net/http/pprof"
  	"os"
  	"os/exec"
  	"runtime"
  	"time"
  )

  const (
  	stopping = byte(iota)
  	working

  	controlURL = "127.0.0.1:33631"
  )

  //Version is aidosd's version. It should be overwritten when building on CI.
  var Version = "2.0.0"

  func main() {
    //
  	flag.Usage = func() {
  		fmt.Fprintf(os.Stderr, "aidosd version %v\n", Version)
  		fmt.Fprintf(os.Stderr, "%s <options>\n", os.Args[0])
  		flag.PrintDefaults()
  	}
  	var child, start, status, stop, doimport, generate bool//, refresh, showSeed, initialize bool
  	flag.BoolVar(&child, "child", false, "start as child")
  	flag.BoolVar(&start, "start", false, "start aidosd (default behaviour)")
  	flag.BoolVar(&status, "status", false, "show status")
  	flag.BoolVar(&stop, "stop", false, "stop aidosd")
    flag.BoolVar(&doimport, "import", false, "Imports a new seed (will prompt user for seed)")
    flag.BoolVar(&generate, "generate", false, "Generates a new seed (will display seed)")

  	flag.Parse()

  	if flag.NFlag() > 1 || flag.NArg() > 0 {
  		flag.Usage()
  		return
  	}
  	if flag.NFlag() == 0 {
  		start = true
  	}

		if child {
  		if err := runChild(); err != nil {
  			panic(err)
  		}
  	}

    if start {
      passwd := []byte(os.Getenv("AIDOSD_PASSWORD"))
      if len(passwd) >= 6 {
        fmt.Println("Using password from evironment variable AIDOSD_PASSWORD")
      }
      aidos.CLImain(passwd,doimport,generate)

      if err := runParent(aidos.Password, os.Args[0]); err != nil {
        panic(err)
      }
      fmt.Println("aidosd has started")
    }

  	if generate || doimport {
  		aidos.CLImain(nil,doimport,generate)
      fmt.Println("Wallet created. You can now start aidosd with the -start parameter.")
      return
  	}
  	if status {
  		stat, err := callStatus()
  		if err != nil {
  			fmt.Println("aidosd is not running")
  			return
  		}
  		switch stat {
  		case working:
  			fmt.Println("aidosd is working")
  		case stopping:
  			fmt.Println("aidosd is stopping")
  		default:
  			fmt.Println("unknown status")
  		}
  	}
  	if stop {
  		stat, err := callStatus()
  		if err != nil || stat == stopping {
  			fmt.Println("aidosd is not running")
  			return
  		}
  		if err := callStop(); err != nil {
  			panic(err)
  		}
  		fmt.Println("aidosd has stopped")
  	}

  }

  func callStatus() (byte, error) {
  	var stat byte
  	err := call("Control.Status", &struct{}{}, &stat)
  	return stat, err
  }

  func callStop() error {
  	return call("Control.Stop", &struct{}{}, &struct{}{})
  }

  //Control is a struct for controlling child.
  type Control struct {
  	status byte
  }

  //Start starts aidosd with password.
  func (c *Control) Start(r *http.Request, args *[]byte, reply *struct{}) error {
    //
    aidos.ParseConf("aidosd.conf")
    //
    if (aidos.Aconf == nil) || len(aidos.Aconf.RPCPort) == 0 {
      log.Fatal("ERROR: aidosd.conf does not contain mandatory fields")
    }

    go func() {
      aidos.ReadAddressesFromFile()
  		for {
  			if _, err := aidos.Walletnotify(aidos.Aconf); err != nil {
  				log.Print(err)
  			}
  			time.Sleep(time.Minute)
  		}
  	}()

  	fmt.Println("starting the aidosd server at port http://0.0.0.0:" + aidos.Aconf.RPCPort)
  	mux := http.NewServeMux()
  	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
  		aidos.Handle(aidos.Aconf, w, r)
  	})
  	go func() {
  		if err := http.ListenAndServe("0.0.0.0:"+aidos.Aconf.RPCPort, mux); err != nil {
  			log.Println(err)
  		}
  	}()
  	c.status = working
  	return nil
  }

  //Stop stops aidosd.
  func (c *Control) Stop(r *http.Request, args *struct{}, reply *struct{}) error {
  	aidos.Exit()
  	c.status = stopping
  	return nil
  }

  //Status returns if aidosd is working or stopping.
  func (c *Control) Status(r *http.Request, args *struct{}, reply *byte) error {
  	*reply = c.status
  	return nil
  }

  func call(method string, args interface{}, ret interface{}) error {
  	url := "http://" + controlURL + "/control"
  	message, err := json.EncodeClientRequest(method, args)
  	if err != nil {
  		return err
  	}
  	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
  	if err != nil {
  		return err
  	}
  	req.Header.Set("Content-Type", "application/json")
  	client := new(http.Client)
  	resp, err := client.Do(req)
  	if err != nil {
  		return fmt.Errorf("Error in sending request to %s. %s", url, err)
  	}
  	defer func() {
  		if err := resp.Body.Close(); err != nil {
  			log.Println(err)
  		}
  	}()

  	return json.DecodeClientResponse(resp.Body, ret)
  }

  func runParent(passwd []byte, oargs ...string) error {
  	args := []string{"-child"}
  	cmd := exec.Command(oargs[0], args...)

  	cmd.Stdout = os.Stdout
  	cmd.Stdin = os.Stdin
  	cmd.Stderr = os.Stderr
  	if err := cmd.Start(); err != nil {
  		return err
  	}
  	time.Sleep(3 * time.Second)
    call("Control.Start", &passwd, &struct{}{})
  	return nil
  }


  func runChild() error {

    flog, err := os.OpenFile("aidosd.daemon.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        fmt.Println("Cannot create log file aidosd.daemon.log ")
        log.Fatal(err)
    }

  	os.Stdout = flog
  	os.Stdin = nil
  	os.Stderr = flog

  	runtime.SetBlockProfileRate(1)
  	go func() {
  		// TODO Remove hardcoded address and port...
  		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
  	}()

  	s := rpc.NewServer()
  	s.RegisterCodec(json.NewCodec(), "application/json")
  	if err := s.RegisterService(new(Control), ""); err != nil {
  		panic(err)
  	}
  	http.Handle("/rpc", s)

  	mux := http.NewServeMux()
  	mux.Handle("/control", s)
  	log.Println("started  control server on aidosd...")
  	return http.ListenAndServe(controlURL, mux)
  }
