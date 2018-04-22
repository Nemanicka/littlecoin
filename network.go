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
  "strconv"
  // "sync/atomic"
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
var isInDivergenceResolutionSession bool

type MessageType int

const (
  Sync        = 1
  B           = 2
  T           = 3
  End         = 4
  Ok          = 5
  InitDivergenceResolving         = 6
  CommonAncestorSearch            = 7
  CommonAncestorResponse          = 8
  SyncProposal = 9
)

type Message struct {
  Type byte
  Size [4]byte
}

type CommonAncestorSearchMessage struct {
  Index int
  Hash  []byte
}

func createBlockMessage(block Block) ([]byte, error) {
  messageBody, err := json.Marshal(block)

  messageType := new(bytes.Buffer)
  err = binary.Write(messageType, binary.LittleEndian, uint8(B))
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

func createSyncProposalMessage() ([]byte, error) {

  resp, err := http.Get("http://ipv4.myexternalip.com/raw")
  if err != nil {
    log.Fatal(err)
  }
  body, _ := ioutil.ReadAll(resp.Body)
  fmt.Println("sending address = ", string(body[:len(body)-1]) + ":" +  os.Getenv("PORT"))
  // messageBody := string(body[:len(body)-1]) + ":" + os.Getenv("PORT")

  messageBody := string(body[:len(body)-1]) + ":" + os.Getenv("PORT")

  messageType := new(bytes.Buffer)
  err = binary.Write(messageType, binary.LittleEndian, uint8(SyncProposal))
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

func createCommonAncestorSearchMessage(index int, hash []byte) ([]byte, error) {
  messageBody, err := json.Marshal(CommonAncestorSearchMessage{index, hash})

  messageType := new(bytes.Buffer)
  err = binary.Write(messageType, binary.LittleEndian, uint8(CommonAncestorSearch))
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


func createCommonAncestorResponseMessage(response string) ([]byte, error) {
  messageBody := []byte(response)

  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(CommonAncestorResponse))
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

func createInitDivergenceResolvingMessage() ([]byte, error) {
  messageBody := "DIVERGENCE"

  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(InitDivergenceResolving))
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
  err := binary.Write(messageType, binary.LittleEndian, uint8(End))
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

func createOkMessage() ([]byte, error) {
  messageBody := []byte("OK")

  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(Ok))
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

  // int sendingIndex = 0;
  IterateBlockchainForward(func(block Block) (bool, error) {
    // sendingIndex += 1
    // fmt.Println(block.Hash, "|", message)
    if bytes.Equal(message, block.Hash) {
      fmt.Println("FOUND")
      sending = true
      // return true, nil
    } else {
      // fmt.Println("Sending?")
      if sending {
        message, err := createBlockMessage(block)
        if err != nil {
          return false, err
        }
        // fmt.Println("sending header = ", message[0], message[1], message[2], message[3], message[4])
        // fmt.Println("sending size  = ", len(message))
        // fmt.Println("sending block  = ", message[4:])
        // fmt.Println("sending hash  = ", block.Hash)
        conn.Write(message)
        err = handleMessages(conn)
        if err != nil {
          return true, err
        }
      }
    }

    return false, nil
  })

  if !sending {
    fmt.Println("Cannot find requested block... Sent divergence resolution proposal")
    m, _ := createInitDivergenceResolvingMessage()
    conn.Write(m)
  } else {
    fmt.Println("end1")
    m, _ := createEndMessage()
    conn.Write(m)
  }
}

// func initDivergenceResolving(conn net.Conn) {
//   // sending := false
//
//   int blockIndex = 0;
//   IterateBlockchainForward(func(block Block) (bool, error) {
//     m, _ := createCommonBlockSearchMessage(blockIndex, block.Hash)
//     conn.Write(m)
//     blockIndex += 1
//
//     idle := time.Now()
//     stop := false
//
//     go func () {
//       for {
//         if stop {
//           break
//         }
//
//         now := time.Now()
//         diff := now.Sub(idle)
//         // fmt.Println(diff)
//         if diff > 5*time.Second  {
//           fmt.Println("Oops, cannot sync you :(")
//           pullConn.Close()
//           break
//         }
//
//         time.Sleep(1)
//       }
//     } ()
//
//     for {
//       idle = time.Now()
//       // go func () {
//       err = handleMessages(pullConn)
//         if err != nil {
//           if err.Error() != "THEEND" && err.Error() != "DIVERGENCE" {
//             fmt.Println("error during message handling")
//           }
//           stop = true
//           break
//         }
//       // } ()
//     }
//
//     if err.Error() == "DIVERGENCE" {
//       isInDivergenceResolutionSession.Store(true)
//       defer isInDivergenceResolutionSession.Store(false)
//
//       blockIndex = 0
//       resolution := nil
//
//       IterateBlockchainBackward(func(block Block) (bool, error) {
//         m := createCommonBlockSearchMessage(blockIndex, block.Hash)
//         pullConn.Write(m)
//         idle = time.Now()
//         stop = false
//         foundCommonAncestor := false
//
//         go func () {
//           for {
//             if stop {
//               break
//             }
//
//             now := time.Now()
//             diff := now.Sub(idle)
//             // fmt.Println(diff)
//             if diff > 5*time.Second  {
//               fmt.Println("Oops, cannot sync you :(")
//               pullConn.Close()
//               break
//             }
//
//             time.Sleep(1)
//           }
//         } ()
//
//         for {
//           idle = time.Now()
//           // go func () {
//           err = handleMessages(pullConn)
//           if err.Error() == "TRYNEXT" {
//
//           } else if err.Error() == "THEEND" {
//             fmt.Println("Failed to find common ancestor block")
//             resolution = errors.New("Failed to find common ancestor block")
//           } else if err.Error() == "FOUNDANCESTORBLOCK" {
//             resolution = errors.New("FOUNDCOMMONANCESTOR")
//           }
//
//           stop = true
//           break
//         }
//
//         return foundCommonAncestor, resolution
//       })
//
//       fmt.Println("Resolution = ", resolution)
//     }
//
//   //
//   //   return false, nil
//   // })
//
//
//
// }

func onBlockReceived(messageBody []byte) error {
  // fmt.Println("got = ", len(messageBody))

  var block Block
  err := json.Unmarshal(messageBody, &block)
  if err != nil {
    fmt.Println("Received block is invalid");
    return err
  }

  fmt.Println("Block received")

  err = AppendToBlockChain(block)
  if err != nil {
    fmt.Println(err)
    return err
  }

  return nil
}

func findCommonAncestor(message []byte, conn net.Conn) error {
  fmt.Println("looking for common ancestor")
  var searchMessage CommonAncestorSearchMessage
  err := json.Unmarshal(message, &searchMessage)
  if err != nil {
    return err
  }

  blockIndex := 0
  found := false

  IterateBlockchainBackward(func(block Block) (bool, error) {
    // fmt.Println("iterating...")
    // fmt.Println(blockIndex, searchMessage.Index)
    if (bytes.Equal(block.Hash, searchMessage.Hash)) {
      found = true

      if (blockIndex > searchMessage.Index) {
        fmt.Println("FOUNDLONGERCHAIN")
        m, _ := createCommonAncestorResponseMessage("FOUNDLONGERCHAIN")
        conn.Write(m)
        // err = errors.New("FOUNDLONGERCHAIN")
      } else if (blockIndex < searchMessage.Index) {
        fmt.Println("FOUNDSHORTERCHAIN")
        m, _ := createCommonAncestorResponseMessage("FOUNDSHORTERCHAIN")
        conn.Write(m)
        // err = errors.New("FOUNDSHORTERCHAIN")
      } else {
        fmt.Println("FOUNDEQUALCHAIN")
        m, _ := createCommonAncestorResponseMessage("FOUNDEQUALCHAIN")
        conn.Write(m)
      }
    }
    // fmt.Println("Increment")
    blockIndex += 1

    return found, nil
  })
  //
  if !found {
    m, _ := createCommonAncestorResponseMessage("TRYNEXT")
    conn.Write(m)
  }

  return err
}

func handleMessages(conn net.Conn) error {
  buff := make([]byte, 5)
  reader := bufio.NewReader(conn)
  _, err := reader.Read(buff)

  // fmt.Println("read ", buff)
  if err != nil {
    // if err != io.EOF {
    fmt.Println("read error: ", err)
    return err
    // }
  }

  var messageType uint8
  b := bytes.NewReader(buff[:1])
  err = binary.Read(b, binary.LittleEndian, &messageType)
  // fmt.Println("type =", messageType)
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
      fmt.Println("You are synced!")
      return errors.New("THEEND")
    case B:
      err = onBlockReceived(messageBody)
      if err != nil {
        return err
      }
      m, _ := createOkMessage()
      conn.Write(m)
    case Ok:
      return nil
    case InitDivergenceResolving:
      return errors.New("DIVERGENCE")
    case CommonAncestorSearch:
      fmt.Println("find common ancestor")
      return findCommonAncestor(messageBody, conn)
    case CommonAncestorResponse:
      return errors.New(string(messageBody))
    case SyncProposal:
      fmt.Println("SYNC PROPOSAL")
      go func() {
        fmt.Println("dialing", string(messageBody))
        newConn, err := net.Dial("tcp", string(messageBody))
        if err != nil {
          fmt.Println("Error dialing")
          return
        }
        pullConn = newConn
        syncData()
      } ()
    // sendBlockchain
    default:
      fmt.Println("Invalid message")
      return errors.New("Invalid message")
  }

  return nil
}

func runServer() error {

  port := os.Getenv("PORT")
  log.Println("Listening on", os.Getenv("PORT"))
  ln, err := net.Listen("tcp", ":" + port)
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
      for {
        err = handleMessages(conn)
        if (err != nil) {
          return
        }
      }
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
    conn, err := net.Dial("tcp", addresses[addressIndex])
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
    address := string(body[:len(body)-1]) + ":" + os.Getenv("PORT")

    addresses = append(addresses, address);

    file.WriteString(address + "\n")
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

  isInDivergenceResolutionSession = false

  if pullConn == nil {
    fmt.Println("No connection to sync from");
    return
  }

  // fmt.Println(len(int))
  // fmt.Println(len(uint))

  blockchainMutex.Lock()
  lastBlock, _ := getLastBlock()
  blockchainMutex.Unlock()
  lastBlockHash := lastBlock.Hash
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
    // go func () {
    err = handleMessages(pullConn)
      if err != nil {
        if err.Error() != "THEEND" && err.Error() != "DIVERGENCE" {
          fmt.Println("error during message handling")
        }
        stop = true
        break
      }
    // } ()
  }

  if err.Error() == "DIVERGENCE" {
    fmt.Println("Start divergence resolution")
    // isInDivergenceResolutionSession.Store(true)
    // defer isInDivergenceResolutionSession.Store(false)

    blockIndex := 0
    resolution := errors.New("")
    resolution = nil

    didle := time.Now()
    dstop := false

    go func () {
      for {
        if dstop {
          break
        }

        now := time.Now()
        diff := now.Sub(idle)
        // fmt.Println(diff)
        if diff > 5*time.Second  {
          fmt.Println("Oops, cannot resolve divergence :(")
          pullConn.Close()
          break
        }

        time.Sleep(1)
      }
    } ()

    IterateBlockchainBackward(func(block Block) (bool, error) {
      m, err := createCommonAncestorSearchMessage(blockIndex, block.Hash)
      if err != nil {
        fmt.Println("error while creating message")
        return true, errors.New("Error")
      }

      // fmt.Println("WRITE", blockIndex)
      pullConn.Write(m)
      blockIndex += 1
      foundCommonAncestor := false



      for {
        didle = time.Now()
        // go func () {
        err = handleMessages(pullConn)
        if err.Error() == "TRYNEXT" {

        // } else if err.Error() == "THEEND" {
          // fmt.Println("Failed to find common ancestor block")
          // resolution = errors.New("Failed to find common ancestor block")
        } else if err.Error() == "FOUNDLONGERCHAIN" {
          resolution = errors.New("FOUNDLONGERCHAIN")
          foundCommonAncestor = true
          dstop = true
        } else if err.Error() == "FOUNDSHORTERCHAIN" {
          resolution = errors.New("FOUNDSHORTERCHAIN")
          foundCommonAncestor = true
          dstop = true
        } else if err.Error() == "FOUNDEQUALCHAIN" {
          resolution = errors.New("FOUNDEQUALCHAIN")
          foundCommonAncestor = true
          dstop = true
        } else {
          fmt.Println("Smth went wrong")
          resolution = errors.New("Error")
        }

        break
      }

      return foundCommonAncestor, nil
    })

    dstop = true
    if resolution == nil {
      fmt.Println("There is no common ancestor, try to delete blockchain.dat file and update client. Sorry(")
    } else if resolution.Error() == "FOUNDLONGERCHAIN" {
      deleteNLastBlocks(blockIndex)
      syncData()
    } else if resolution.Error() == "FOUNDSHORTERCHAIN" {
      m, _ := createSyncProposalMessage()
      pullConn.Write(m)
    } else if resolution.Error() == "FOUNDEQUALCHAIN" {
      fmt.Println("Divergence with an equal chain length, wait until one become longer")
        // m, _ := createEndMessage
        // pullConn.Write(m)
    }
    fmt.Println("Resolution = ", resolution)
  }
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
  fmt.Print("Port: ")
  bytes, _  = buf.ReadBytes('\n')
  portBytes := string(bytes[:len(bytes) - 1])

  if err != nil {
    fmt.Println(err)
  } else {
    ip := net.ParseIP(ipstr)
    if ip.To4() == nil {
      fmt.Println("This address is not valid, try again")
      return
    }

    port, err2 := strconv.Atoi(string(portBytes))
    if err2 != nil {
      fmt.Println(err2)
      return
    }

    if (port < 1500 || port > 50000) {
      fmt.Println("Port should be in range from 1500 to 50000")
    }

    address := ipstr + ":" + string(portBytes)

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

    addresses = append(addresses, address);
    file.WriteString(address + "\n")

    // emitMessage(ipstr, Message{"GET", url.Values{}, "updates"})
  }
}
