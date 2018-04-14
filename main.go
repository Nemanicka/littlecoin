package main

import (
  "log"
  // "net"
  "os"
  // "io"
  // "time"
  "bufio"
  "fmt"
  "github.com/davecgh/go-spew/spew"
  // "github.com/gorilla/mux"
  "github.com/joho/godotenv"
)

// func makeMuxRouter() http.Handler {
//   muxRouter := mux.NewRouter()
//   return muxRouter
// }


func showHelp() {
  fmt.Println("help             show this message");
  fmt.Println("balance          show your balance");
  fmt.Println("peers            show list of all available peers");
  fmt.Println("transactions     show list of your transactions");
  fmt.Println("send             send money... follow instructions");
  fmt.Println("pending          show your pending transactions");
  fmt.Println("mine             start mining");
  fmt.Println("addbuddy         add certain peer to your address book");
}

func processInput(cmd string) {
  switch cmd {
  case "help":
    showHelp();
  case "transactions":
    showTransactions();
  case "send":
    send();
  case "mine":
    if mining {
      mining = false
      fmt.Println("Stopping mining...")
    } else {
      fmt.Println("Start mining, to stop, type this command once again")
      mining = true
      go func() {
        Mine()
      } ()
    }
  case "pending":
    txsout, txsin := getPendingTransactions()
    spew.Dump(txsout)
    spew.Dump(txsin)
  case "balance":
    confirmedBalance, unconfirmedBalance := getBalance()
    fmt.Println("Your confirmed balance: ", confirmedBalance, " ultramegacoins")
    fmt.Println("Your pending   balance:   ", unconfirmedBalance, " ultramegacoins")
  case "addbuddy":
    addBuddy()
  case "peers":
    showAddresses()
  // case "sync":
    // syncData()
  default:
    fmt.Println("Unknown command %s.\nType 'help' to get full command list", cmd)
  }
}

func getInput() {
  buf := bufio.NewReader(os.Stdin)
  fmt.Print("> ")
  command, err := buf.ReadBytes('\n')
  if err != nil {
    fmt.Println(err)
  } else {
    processInput(string(command[:len(command) - 1]))
  }

  defer getInput()
}

func main() {
  // Load environmental vars - address, etc
  err := godotenv.Load()
  if err != nil {
    log.Fatal(err)
  }

  // Load blockchain.dat and awallet.dat
  loadFiles()

  blockchainMutex.Lock()
  lastBlock, _ := getLastBlock()
  blockchainMutex.Unlock()

  // If blockchain file is empty, create genesis block
  if len(lastBlock.Hash) == 0 {
    genesisBlock := CreateGenesisBlock()
    fmt.Println("Created genesis block")
    err = AppendToBlockChain(genesisBlock)

    if err != nil {
      log.Fatal(err)
    }
  }

  // If wallet file is empty, create private/public keys
  if len(pubKey)==0 {
    err = CreateWalllet()

    if err != nil {
      log.Fatal(err)
    }
  }

  initAddresses()

  initNetwork.Add(1)

  go func() {
    runServer()
  } ()

  initNetwork.Wait()

  connect()

  syncData()

  getInput()
}
