package main

import (
  "log"
  "os"
  "os/signal"
  "time"
  "syscall"
  "sync"
  "net/http"
  "github.com/8legd/go-tigertonic"
)


func main() {

  // Start the HTTP server
  var handler http.HandlerFunc
  handler = func(w http.ResponseWriter, r *http.Request) {
    time.Sleep(10 * time.Second)
    w.WriteHeader(http.StatusOK)
    return
  }
  server := tigertonic.NewServer(":8000", handler)

  // We start the server in a seperate goroutine to the main goroutine
  // so we can handle stopping the server gracefully
  go func() {
    err := server.ListenAndServe()
    if nil != err {
      log.Println(err)
    }
  }()

  var wg sync.WaitGroup
  wg.Add(1000)
  log.Println("Starting Experiment At",time.Now())
  for i := 0; i<1000; i++ {
    go func() {
      defer wg.Done()
      res, err := http.Get("http://localhost:8000/")
      if err != nil {
        log.Println(err)
      } else {
        log.Println("Reading Response ",res.StatusCode)
      }
    }()
  }
  wg.Wait()
  log.Println("Stopping Experiment At",time.Now())

  // By creating an un-buffered chanel the main goroutine will in effect block here
  // waiting for an os.Signal at which point we can stop the server gracefully
  ch := make(chan os.Signal)
  signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
  s := <-ch
  log.Println("Closing server on OS signal: ",s)
  server.Close() // this closes all connections



}
