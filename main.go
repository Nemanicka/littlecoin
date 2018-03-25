package main

import (
  "log"
  "net/http"
  "os"
  "time"
  "bufio"
  "fmt"
  "github.com/davecgh/go-spew/spew"
  "github.com/gorilla/mux"
  "github.com/joho/godotenv"
)

func run() error {
  mux := makeMuxRouter()
  httpAddr := os.Getenv("ADDR")
  log.Println("Listening on", os.Getenv("ADDR"))
  s:= &http.Server{
    Addr           : ":" + httpAddr,
    Handler        : mux,
    ReadTimeout    : 10 * time.Second,
    WriteTimeout   : 10 * time.Second,
    MaxHeaderBytes : 1 << 20,
  }

  if err := s.ListenAndServe(); err != nil {
    return err
  }

  return nil
}

func makeMuxRouter() http.Handler {
  muxRouter := mux.NewRouter()
  return muxRouter
}


func showHelp() {
  fmt.Println("help             show this message");
  fmt.Println("balance          show your balance");
  fmt.Println("peers            show list of all available peers");
  fmt.Println("transactions     show list of your transactions");
  fmt.Println("send             [Amount][Address] send money");
  fmt.Println("pending          show your pending transactions");
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
    fmt.Println("Your confirmed balance: ", confirmedBalance, " ultramegacoins");
    fmt.Println("Your pending   balance:   ", unconfirmedBalance, " ultramegacoins");
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

  // Listen to the user's input
  go func() {
    getInput()
  } ()

  // Listen to network
  log.Fatal(run())
}
