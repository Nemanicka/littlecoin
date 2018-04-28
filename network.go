package main

import (
  "log"
  "net/http"
  "os"
  "time"
  "net"
  "bufio"
  "fmt"
  "io/ioutil"
  "io"
  "encoding/json"
  "encoding/binary"
  "strconv"
  "errors"
  "bytes"
  "math/rand"
  "sync"
)

var networkMutex sync.Mutex
var addresses []string
var addressesFileName = "addresses.dat"
var pendingRequests sync.WaitGroup
var initNetwork sync.WaitGroup
var recipients map[string]net.Conn
var pullConn net.Conn
var isInDivergenceResolutionSession bool
var syncing bool

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

func createMessage(mBody []byte, mType uint8) ([]byte, error) {
  messageType := new(bytes.Buffer)
  err := binary.Write(messageType, binary.LittleEndian, uint8(mType))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  messageLength := new(bytes.Buffer)
  err = binary.Write(messageLength, binary.LittleEndian, uint32(len(mBody)))
  if err != nil {
    fmt.Println("failed", err)
    return []byte{}, err
  }

  message := append(messageType.Bytes(), messageLength.Bytes()...)
  message = append(message, mBody...)
  return message, nil
}

func propagateBlock(block Block) {
  messageBody, _ := json.Marshal(block)
  message, err := createMessage(messageBody, uint8(B))

  if err != nil {
    fmt.Println("Error during transaction propagation")
  }

  networkMutex.Lock()
  for _, conn := range recipients {
    conn.Write(message)
  }
  networkMutex.Unlock()
}

func propagateTransaction(transaction Transaction) {
  messageBody, _ := json.Marshal(transaction)
  message, err := createMessage(messageBody, uint8(T))

  if err != nil {
    fmt.Println("Error during transaction propagation")
  }

  networkMutex.Lock()
  for _, conn := range recipients {
    conn.Write(message)
  }
  networkMutex.Unlock()
}

func createBlockMessage(block Block) ([]byte, error) {
  messageBody, err := json.Marshal(block)
  if err != nil {
    return []byte{}, err
  }

  return createMessage(messageBody, uint8(B))
}

func createSyncProposalMessage() ([]byte, error) {

  resp, err := http.Get("http://ipv4.myexternalip.com/raw")
  if err != nil {
    log.Fatal(err)
  }
  body, _ := ioutil.ReadAll(resp.Body)
  fmt.Println("sending address = ", string(body[:len(body)-1]) + ":" +  os.Getenv("PORT"))
  // messageBody := string(body[:len(body)-1]) + ":" + os.Getenv("PORT")

  // messageBody := string(body[:len(body)-1]) + ":" + os.Getenv("PORT")
  messageBody := "127.0.0.1:" + os.Getenv("PORT")
  return createMessage([]byte(messageBody), SyncProposal)
}

func createCommonAncestorSearchMessage(index int, hash []byte) ([]byte, error) {
  messageBody, _ := json.Marshal(CommonAncestorSearchMessage{index, hash})
  return createMessage(messageBody, CommonAncestorSearch)
}

func createCommonAncestorResponseMessage(response string) ([]byte, error) {
  messageBody := []byte(response)
  return createMessage(messageBody, CommonAncestorResponse)
}

func createInitDivergenceResolvingMessage() ([]byte, error) {
  messageBody := []byte("DIVERGENCE")
  return createMessage(messageBody, InitDivergenceResolving)
}

func createEndMessage() ([]byte, error) {
  messageBody := []byte("THEEND")
  return createMessage(messageBody, End)
}

func createOkMessage() ([]byte, error) {
  messageBody := []byte("OK")
  return createMessage(messageBody, Ok)
}

func sendBlockchain(message []byte, conn net.Conn) {
  sending := false

  // int sendingIndex = 0;
  IterateBlockchainForward(func(block Block, index int) (bool, error) {
    // sendingIndex += 1
    fmt.Println(block.Hash, "|", message)
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

        conn.Write(message)
        _, err = handleMessages(conn)
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

func onTransactionReceived(messageBody []byte) error {
  PendingTransactionsMutex.Lock()
  var transaction Transaction
  err := json.Unmarshal(messageBody, &transaction)
  if err != nil {
    fmt.Println("Received transaction is invalid, cannot unmarshal");
    return err
  }

  for _, tx := range PendingTxs {
    if bytes.Equal(tx.Id, transaction.Id) {
      // Do nothing
      return nil
    }
  }

  PendingTransactionsMutex.Unlock()
  propagateTransaction(transaction)
  OnPendingTxsAdded(transaction)
  return nil
}

func cleanTransactions(block Block, txs *[]Transaction) {
  for _, tx := range block.Txs {
    deleteIndex := -1
    for index, ptx := range *txs {
      if bytes.Equal(tx.Id, ptx.Id) {
        deleteIndex = index
      }
    }

    if deleteIndex != -1 {
      if len(*txs) == 1 {
        var empty []Transaction
        (*txs) = empty
      } else {
        *txs = append((*txs)[:deleteIndex], (*txs)[deleteIndex+1:]...)
      }
    }
  }
}

func onBlockReceived(messageBody []byte) error {
  // fmt.Println("got = ", len(messageBody))

  var block Block
  err := json.Unmarshal(messageBody, &block)
  if err != nil {
    fmt.Println("Received block is invalid");
    return err
  }

  fmt.Println("Block received")
  if !syncing {
    blockchainMutex.Lock()
    defer blockchainMutex.Unlock()
  }
  err = AppendToBlockChain(block)
  if err != nil {
    fmt.Println(err)
    return err
  }

  PendingTransactionsMutex.Lock()
  defer PendingTransactionsMutex.Unlock()

  cleanTransactions(block, &PendingTxs)
  cleanTransactions(block, &VerifiedPendingTxs)
  return nil
}

func findCommonAncestor(message []byte, conn net.Conn) (string, error) {
  fmt.Println("looking for common ancestor")
  var searchMessage CommonAncestorSearchMessage
  err := json.Unmarshal(message, &searchMessage)
  if err != nil {
    return "", err
  }

  blockIndex := 0
  found := false

  IterateBlockchainBackward(func(block Block, index int) (bool, error) {
    // fmt.Println("iterating...")
    // fmt.Println(blockIndex, searchMessage.Index)
    if (bytes.Equal(block.Hash, searchMessage.Hash)) {
      found = true

      if (blockIndex > searchMessage.Index) {
        if searchMessage.Index == 0 {
          fmt.Println("sending blockchain")
        } else {
          fmt.Println("FOUNDLONGERCHAIN")
          m, _ := createCommonAncestorResponseMessage("FOUNDLONGERCHAIN")
          conn.Write(m)
        }
      } else if (blockIndex < searchMessage.Index) {
        fmt.Println("FOUNDSHORTERCHAIN")
        m, _ := createCommonAncestorResponseMessage("FOUNDSHORTERCHAIN")
        conn.Write(m)
      } else {
        if searchMessage.Index == 0 {
          m, _ := createEndMessage()
          conn.Write(m)
        } else {
          fmt.Println("FOUNDEQUALCHAIN")
          m, _ := createCommonAncestorResponseMessage("FOUNDEQUALCHAIN")
          conn.Write(m)
        }
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

  if searchMessage.Index == 0 && found {
    sendBlockchain(searchMessage.Hash, conn)
  }

  return "OK", err
}

func handleMessages(conn net.Conn) (string, error) {
  buff := make([]byte, 5)
  reader := bufio.NewReader(conn)
  _, err := reader.Read(buff)

  // fmt.Println("read ", buff)
  if err != nil {
    // if err != io.EOF {
    fmt.Println("read error: ", err)
    return "", err
    // }
  }

  var messageType uint8
  b := bytes.NewReader(buff[:1])
  err = binary.Read(b, binary.LittleEndian, &messageType)
  // fmt.Println("type =", messageType)
  if err != nil {
      fmt.Println("binary.Read failed:", err)
      return "", err
  }
  fmt.Println("type = ", messageType)

  var messageSize uint32
  b = bytes.NewReader(buff[1:])
  err = binary.Read(b, binary.LittleEndian, &messageSize)
  if err != nil {
      fmt.Println("binary.Read failed:", err)
      return "", err
  }
  fmt.Println("size = ", messageSize)

  messageBody := make([]byte, messageSize)
  _, err = reader.Read(messageBody)
  if err != nil {
    if err != io.EOF {
      fmt.Println("read error: ", err)
      return "", err
    }
  }

  switch messageType {
    case Sync:
      sendBlockchain(messageBody, conn)
    case End:
      fmt.Println("You are synced!")
      return "THEEND", nil
    case B:
      err = onBlockReceived(messageBody)
      fmt.Println("err", err)
      if err != nil {
        fmt.Println(err)
        return "", err
      }
      m, _ := createOkMessage()
      conn.Write(m)
    case T:
      onTransactionReceived(messageBody)
    case Ok:
      return "OK", nil
    case InitDivergenceResolving:
      return "DIVERGENCE", nil
    case CommonAncestorSearch:
      fmt.Println("find common ancestor")
      return findCommonAncestor(messageBody, conn)
    case CommonAncestorResponse:
      fmt.Println("got", string(messageBody))
      return string(messageBody), nil
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
      return "", errors.New("Invalid message")
  }
  // fmt.Println("return nil")
  return "OK", nil
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
        _, err = handleMessages(conn)
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

  recipients = make(map[string]net.Conn)

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
      recipients[addresses[addressIndex]] = conn
      fmt.Println("Connected to ", addresses[addressIndex])
      break
    }
  }

  fmt.Println("Choosing recipients")

  addressIndex = rand.Intn(len(addresses) - 1) + 1
  head         = addressIndex
  for {
    if len(recipients) == (len(addresses) - 1) ||
       len(recipients) >= 10 {
      return
    }

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
        fmt.Println("There is no more peers to choose as recipients, try later")
        break;
      }
    } else {
      recipients[addresses[addressIndex]] = conn
      fmt.Println("Recipient added: ", addresses[addressIndex])

      if addressIndex == len(addresses) - 1 {
        addressIndex = 1
      } else {
        addressIndex += 1
      }

    }
  }
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

func syncData() {
  syncing = true
  networkMutex.Lock()
  if pullConn == nil {
    fmt.Println("No connection to sync from");
    networkMutex.Unlock()
    return
  }

  didle := time.Now()
  dstop := false

  /// If connection is idle for more then 5 sec, stop it
  go func () {
    for {
      if dstop {
        break
      }

      now := time.Now()
      diff := now.Sub(didle)
      // fmt.Println(diff)
      if diff > 5*time.Second  {
        fmt.Println("Oops, cannot resolve divergence :(")
        pullConn.Close()
        break
      }

      time.Sleep(1)
    }
  } ()

  blockIndex := 0
  handlingResult := "ERROR"
  iteratingResult := IterateBlockchainBackward(func(block Block, index int) (bool, error) {
    blockIndex = index
    m, err := createCommonAncestorSearchMessage(index, block.Hash)
    if err != nil {
      fmt.Println("error while creating message")
      return true, errors.New("Error")
    }

    pullConn.Write(m)

    for {
      didle = time.Now()
      handlingResult, err = handleMessages(pullConn)
      if handlingResult == "FOUNDLONGERCHAIN"  ||
         handlingResult == "FOUNDSHORTERCHAIN" ||
         handlingResult == "FOUNDEQUALCHAIN"   ||
         handlingResult == "THEEND" {
        // Stop iteration
        return true, err
      }

      if handlingResult == "TRYNEXT" {
        break
      }

      if handlingResult != "OK" {
        return true, err
      }
    }

    return false, err
  })

  /// Stop connection listening routine
  dstop = true

  if iteratingResult != nil {
    fmt.Println("Iteration ends with error")
  }

  if handlingResult == "ERROR" {
    fmt.Println("There is no common ancestor, try to delete blockchain.dat file and update client. Sorry(")
  } else if handlingResult == "FOUNDLONGERCHAIN" {
    fmt.Println("index ", blockIndex)
    if isInDivergenceResolutionSession {
      fmt.Println("do nothing, we've been through this with no luck")
      isInDivergenceResolutionSession = false
    } else {
      isInDivergenceResolutionSession = true
      if blockIndex != 0 {
        fmt.Println("delete ", blockIndex)
        deleteNLastBlocks(blockIndex)
      }
      // syncing = false
      // networkMutex.Unlock()
      defer syncData()
      // return
    }
  } else if handlingResult == "FOUNDSHORTERCHAIN" {
    m, _ := createSyncProposalMessage()
    pullConn.Write(m)
  } else if handlingResult == "FOUNDEQUALCHAIN" {
    fmt.Println("Divergence with an equal chain length, wait until one become longer")
  } else if handlingResult == "THEEND" {
      // pullConn.Write(m)
  }
  syncing = false
  networkMutex.Unlock()
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
