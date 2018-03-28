package main

import (
  "log"
  "net/http"
  "os"
  //"time"
  "bufio"
  "fmt"
  "io/ioutil"
  //"github.com/davecgh/go-spew/spew"
  //"github.com/gorilla/mux"
  //"github.com/joho/godotenv"
  //"crypto/sha256"
  //"encoding/json"
  //"os"
  //"bufio"
  //"errors"
  //"github.com/serverhorror/rog-go/reverse"
  "sync"
)

var networkMutex sync.Mutex
var addresses []string
var addressesFileName = "addresses.dat"


func loadAddresses() {
  networkMutex.Lock()
  defer networkMutex.Unlock()

  if _, err := os.Stat(addressesFileName); err == nil {
    var file, _ = os.OpenFile(addressesFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
      line := scanner.Text()
      addresses = append(addresses, line)
    }
  }
}

func initAddresses() {
  if len(addresses) == 0 {
    resp, err := http.Get("http://ipv4.myexternalip.com/raw")
    if err != nil {
      log.Fatal(err)
    }
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(body))

    networkMutex.Lock()
    defer networkMutex.Unlock()

    var file, _ = os.OpenFile(addressesFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
    defer file.Close()

    file.WriteString(string(body))
  }
}

func pull() {
  networkMutex.Lock()
  defer networkMutex.Unlock()

  if len(addresses) < 2 {
    fmt.Println("\n-------------------\n")
    fmt.Println("Hello, newbie, I cannot sync you with others, because there's no 'others', but you can add one or two by typing 'addbuddy'")
    fmt.Println("\n-------------------\n")
  }
}
