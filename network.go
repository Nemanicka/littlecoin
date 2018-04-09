package main

import (
  "log"
  "net/http"
  "os"
  "time"
  "net"
  // "net/url"
  "bufio"
  "fmt"
  "io/ioutil"
  "io"
  "encoding/json"
  "encoding/binary"
  //"github.com/davecgh/go-spew/spew"
  //"github.com/gorilla/mux"
  //"github.com/joho/godotenv"
  //"crypto/sha256"
  //"encoding/json"
  //"os"
  //"bufio"
  "errors"
  //"github.com/serverhorror/rog-go/reverse"
  // "math"
  "bytes"
  "math/rand"
  "sync"
  // "strings"
)

var networkMutex sync.Mutex
var addresses []string
var addressesFileName = "addresses.dat"
var pendingRequests sync.WaitGroup
var initNetwork sync.WaitGroup
var recipients map[string]net.Conn
var pullConn net.Conn

type MessageType int

const (
  Sync        = 1
  B           = 2
  T           = 3
  End         = 4
)

type Message struct {
  Type byte
  Size [4]byte
}

func createBlockMessage(block Block) ([]byte, error) {
  messageBody, err := json.Marshal(block)

  messageType := new(bytes.Buffer)
  err = binary.Write(messageType, binary.LittleEndian, uint8(End))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  messageLength := new(bytes.Buffer)
  err = binary.Write(messageLength, binary.LittleEndian, uint32(len(messageBody)))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  message := append(messageType.Bytes(), messageLength.Bytes()...)
  message = append(message, messageBody...)
  return message, nil
}

func createEndMessage() ([]byte, error) {
  messageBody := []byte("THEEND")

  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(B))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  messageLength := new(bytes.Buffer)
  err = binary.Write(messageLength, binary.LittleEndian, uint32(len(messageBody)))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  message := append(messageType.Bytes(), messageLength.Bytes()...)
  message = append(message, messageBody...)
  return message, nil
}

func sendBlockchain(message []byte, conn net.Conn) {
  sending := false

  IterateBlockchainForward(func(block Block) (bool, error) {
    if bytes.Equal(message, block.Hash) {
      fmt.Println("FOUND")
      sending = true
      // return true, nil
    } else {
      if sending {
        message, err := createBlockMessage(block)
        if err != nil {
          return false, err
        }

        conn.Write(message)
      }
    }

    return false, nil
  })

  m, _ := createEndMessage()

  conn.Write(m)
}

func handleMessages(conn net.Conn) error {
  buff := make([]byte, 5)
  reader := bufio.NewReader(conn)
  _, err := reader.Read(buff)
  fmt.Println("read ", buff)
  if err != nil {
    if err != io.EOF {
      fmt.Println("read error: ", err)
      return err
    }
  }

  var messageType uint8
  b := bytes.NewReader(buff[:1])
  err = binary.Read(b, binary.LittleEndian, &messageType)
  if err != nil {
      fmt.Println("binary.Read failed:", err)
      return err
  }
  // fmt.Println("type = ", messageType)

  var messageSize uint32
  b = bytes.NewReader(buff[1:])
  err = binary.Read(b, binary.LittleEndian, &messageSize)
  if err != nil {
      fmt.Println("binary.Read failed:", err)
      return err
  }
  // fmt.Println("size = ", messageSize)

  messageBody := make([]byte, messageSize)
  _, err = reader.Read(messageBody)
  if err != nil {
    if err != io.EOF {
      fmt.Println("read error: ", err)
      return err
    }
  }
  switch messageType {
    case Sync:
      sendBlockchain(messageBody, conn)
    case End:
      return errors.New("THEEND")
    // sendBlockchain
    default: return errors.New("Invalid message")
  }

  return nil
}

func runServer() error {
  ln, err := net.Listen("tcp", ":12321")
  if (err != nil) {
    return err
  }

  initNetwork.Done()

  for {
    conn, err := ln.Accept()
    if err != nil {
      continue
    }

    go func () {
      handleMessages(conn)
    } ()
  }

  return nil
}

func connect() {
  networkMutex.Lock()
  defer networkMutex.Unlock()

  if len(addresses) < 2 {
    fmt.Println("\n-------------------\n")
    fmt.Println("Hello, newbie, I cannot sync you with others, because there's no 'others', but you can add one or two by typing 'addbuddy'")
    fmt.Println("\n-------------------\n")
    return
  }

  addressIndex := rand.Intn(len(addresses) - 1) + 1
  head         := addressIndex
  for {
    // get next address
    fmt.Println("Dialing...")
    conn, err := net.Dial("tcp", addresses[addressIndex] + ":12321")
    if (err != nil) {
      fmt.Println("Cannot dial " + addresses[addressIndex])
      if addressIndex == len(addresses) - 1 {
        addressIndex = 1
      } else {
        addressIndex += 1
      }

      if addressIndex == head {
        fmt.Println("There is no peers up right now, try later")
        break;
      }

    } else {
      pullConn = conn
      fmt.Println("Connected to ", addresses[addressIndex])
      break
    }
  }
  // pendingRequests.Add(1)

}

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
    addresses = append(addresses, string(body));

    file.WriteString(string(body))
  }
}

func emitMessage(address string, message Message) {
  // defer pendingRequests.Done()
  // uri := "http://" + address + ":12321/" + message.Path;
  //
  // fmt.Println("asking for updates ", uri)
  //
  // if (message.Method == "GET") {
  //   http.Get(uri)
  // } else if (message.Method == "POST") {
  //   http.PostForm(uri, message.Body)
  // }
}

func emitMessages(message Message) {
  // recipientsNum := int(math.Min(float64(10), float64(len(addresses) - 1)))

  // for i := 0; i < recipientsNum; i++ {
  //   address := rand.Intn(len(addresses) - 1) + 1
  //   pendingRequests.Add(1)
  //   go emitMessage(addresses[address], message)
  // }
}

func syncData() {
  networkMutex.Lock()
  defer networkMutex.Unlock()

  if pullConn == nil {
    fmt.Println("No connection to sync from");
    return
  }

  // fmt.Println(len(int))
  // fmt.Println(len(uint))


  lastBlock, _ := getLastBlock()
  lastBlockHash := lastBlock.Hash
  // buf := make([]byte, binary.MaxVarintLen64)
  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(Sync))
  if err != nil {
    fmt.Println("failed", err)
  }

  messageLength := new(bytes.Buffer)
  err = binary.Write(messageLength, binary.LittleEndian, uint32(len(lastBlockHash)))
  if err != nil {
    fmt.Println("failed", err)
  }

  message := append(messageType.Bytes(), messageLength.Bytes()...)

  // fmt.Println("len = ", buf.Bytes())
  // var b bytes.Buffer
  // fmt.Println("bytes = ", len(b.Bytes()))
	// fmt.Fprint(&b, Sync, 0)
  // n := binary

	// return b.Bytes(), nil

  // fmt.Println("hashlen = ", len(lastBlockHash))
  // bytes, _ := json.Marshal(message)
  fmt.Println("message = ", len(message))

  if len(message) != 5 {
    fmt.Println("Not going to send this message, too long", len(message), "bytes");
    return
  }

  message = append(message, lastBlockHash...)

  // bytes = append(bytes, lastBlockHash...)

  pullConn.Write(message)
  if (err != nil) {
    fmt.Println("error!");
  }

  idle := time.Now()
  stop := false

  go func () {
    for {
      if stop {
        break
      }

      now := time.Now()
      diff := now.Sub(idle)
      // fmt.Println(diff)
      if diff > 5*time.Second  {
        fmt.Println("Oops, cannot sync you :(")
        pullConn.Close()
        break
      }

      time.Sleep(1)
    }
  } ()

  for {
    idle = time.Now()
    err = handleMessages(pullConn)
    if err != nil {
      stop = true
      break
    }
  }

  fmt.Println("You are synced!")

  // if len(addresses) < 2 {
  //   fmt.Println("\n-------------------\n")
  //   fmt.Println("Hello, newbie, I cannot sync you with others, because there's no 'others', but you can add one or two by typing 'addbuddy'")
  //   fmt.Println("\n-------------------\n")
  // }

  // emitMessages(Message{"GET", url.Values{}, "updates"});
  // pendingRequests.Wait()
}

func showAddresses() {
  if len(addresses) == 0 {
    fmt.Println("addresses are not loaded")
    return
  }

  fmt.Println(addresses[0], "<- your address")

  if len(addresses) == 1 {
    fmt.Println("You have no peers now, but you can add one by typing 'addbuddy'")
    return
  }

  for _, address := range addresses[1:] {
    fmt.Println(address);
  }
}

func addBuddy() {
  buf := bufio.NewReader(os.Stdin)
  fmt.Print("Address: ")
  bytes, err := buf.ReadBytes('\n')
  ipstr := string(bytes[:len(bytes) - 1])

  if err != nil {
    fmt.Println(err)
  } else {
    ip := net.ParseIP(ipstr)
    if ip.To4() == nil {
      fmt.Println("This address is not valid, try again")
      return
    }

    for _, address := range addresses {
      if ipstr == address {
        fmt.Println("This address is already in yout list")
        return
      }
    }

    networkMutex.Lock()
    defer networkMutex.Unlock()

    var file, _ = os.OpenFile(addressesFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
    defer file.Close()

    addresses = append(addresses, ipstr + "\n");
    file.WriteString(ipstr)

    // emitMessage(ipstr, Message{"GET", url.Values{}, "updates"})
  }
}
