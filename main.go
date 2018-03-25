package main

import (
  "crypto/rand"
  "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/x509"
  "encoding/json"
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

func processInput (cmd string) {
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

func getInput () {
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

func main () {
  err := godotenv.Load()
  if err != nil {
    log.Fatal(err)
  }

  loadFiles()

  go func() {
    log.Print("len = ", len(lastBlock.Hash))
    if len(lastBlock.Hash) == 0 {
      var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer blockfile.Close()
      /// Hardcode genesis block
      txin  := TXIN{[]byte{}, []byte{}}
      txout := TXOUT{"zZvNvCqtvZ3FhUjO+QjoiBQoj+Pgj5GNJDO7z2HifSxGvDfjKHuutUQWLCHifFyXfYNss/LAxYschi3oLLnKww==", 50}
      tx    := Transaction{[]byte{}, []TXIN{txin}, []TXOUT{txout}}
      tx.Id  = tx.Hash()
      txs   := []Transaction{tx}
      genesisBlock := Block{"10.03.2018 easy peasy lemon squeezy", []byte{},
                            []byte{'G', 'E', 'N', 'E', 'S', 'I', 'S'}, txs,
                            []byte{'N', 'O', 'N', 'C', 'E'}}
      genesisBlock.Hash = genesisBlock.HashBlock()
      spew.Dump(genesisBlock)
      // Blockchain = append(Blockchain, genesisBlock)
      str, err2 := json.Marshal(genesisBlock)
      if err2 != nil {
        log.Fatal(err)
        return
      }
      blockfile.WriteString(string(str) + "\n")
    }

    if len(pubKey)==0 {
      curve := elliptic.P256()
    	private, err := ecdsa.GenerateKey(curve, rand.Reader)
    	if err != nil {
    		log.Panic(err)
    	}

      var wallet, _    = os.OpenFile(walletFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer wallet.Close()

    	pubKey = append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
      str, _ := x509.MarshalECPrivateKey(private)
      wallet.Write(str)
    }

  } ()

  go func() {
    getInput()
  } ()

  log.Fatal(run())
}
