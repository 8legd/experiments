package main

import (
  "io/ioutil"
  "log"
  "net"
  "net/http"
  "net/url"
  "os"
  "os/signal"
  "strconv"
  "sync"
  "sync/atomic"
  "syscall"
  "time"
  "github.com/8legd/go-tigertonic"
)

var (
  requestCounter int64
  startedExperiment time.Time
  stoppedExperiment time.Time
  closing bool
  wg sync.WaitGroup
)

// Experiment 1 starts a HTTP server with a basic handler and simulates 10 seconds processing time
// for each request so we can easily overload the server
func main() {

  closing = false

  var handler http.HandlerFunc
  handler = func(w http.ResponseWriter, r *http.Request) {
    // Retrieve the counter from the request.
    var counter string
    counter = r.FormValue("counter")
    //log.Println("Begin Processing Request",counter,"At",time.Now())
    // Sanity checking
    if counter == "" {
      http.Error(w, "you must specify a counter for each request", http.StatusBadRequest)
      log.Println("Completed Processing Request With Error","At",time.Now())
      return
    }
    time.Sleep(10 * time.Second)
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Write([]byte(counter))
    //log.Println("Completed Processing Request",counter,"At",time.Now())
    return
  }
  server := tigertonic.NewServer(":8000", handler)

  // We start the server in a seperate goroutine to the main goroutine
  // so we can handle stopping the server gracefully
  go func() {
    err := server.ListenAndServe()
    if nil != err {
      if opError, ok := err.(*net.OpError); ok &&
        (opError.Err.Error() == "use of closed network connection") &&
        closing {
        // This is acceptable if we are closing the server
      } else {
        log.Println("Unexpected Server Error",err)
      }
    }
  }()

  // Send 10000 simultaneous requests to overload the server (well does on my laptop!)
  var wg sync.WaitGroup
  wg.Add(10000)
  startedExperiment = time.Now()
  log.Println("Starting Experiment At",startedExperiment)
  for i := 0; i<10000; i++ {
    go func() {
      defer wg.Done()
      atomic.AddInt64(&requestCounter, 1)
      //log.Println("Sending Request",requestCounter,"At",time.Now())
      res, err := http.PostForm("http://localhost:8000/",url.Values{"counter": {strconv.FormatInt(requestCounter,10)}})
      if err != nil {
        log.Println("Error Sending Request",requestCounter,"At",time.Now(),err)
        log.Println(err)
      } else {
        defer res.Body.Close()
        _, err := ioutil.ReadAll(res.Body)
        if err != nil {
          log.Println("Error Reading Response","At",time.Now(),err)
          log.Println(err)
        } else {
          atomic.AddInt64(&requestCounter, -1)
          //log.Println("Reading Response",string(body),"At",time.Now(),res.StatusCode)
        }
      }
    }()
  }
  wg.Wait()
  stoppedExperiment = time.Now()
  log.Println("Stopping Experiment At",stoppedExperiment,"Successfully Read",10000-requestCounter,"Requests In",stoppedExperiment.Sub(startedExperiment))


  // By creating an un-buffered chanel the main goroutine will in effect block here
  // waiting for an os.Signal at which point we can stop the server gracefully
  ch := make(chan os.Signal)
  signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
  s := <-ch
  log.Println("Closing server on OS signal: ",s)
  closing = true
  server.Close() // this closes all connections

}
