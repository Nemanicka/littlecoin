package main


import (
  "crypto/sha256"
  // "crypto/rsa"
  "crypto/rand"
  "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/x509"
  "encoding/base64"
  // "encoding/hex"
  "encoding/json"
  // "io"
  "log"
  "net/http"
  "os"
  "time"
  "bufio"
  "io/ioutil"
  "math/big"
  "fmt"
  // "bytes"
  "github.com/davecgh/go-spew/spew"
  "github.com/gorilla/mux"
  "github.com/joho/godotenv"
  // "github.com/dustin/go-hashset"
)

type TXOUT struct {
  Address string;
  Amount int;
  //...
}
//
func (txout TXOUT) Hash() string {
  txoutBytes := append([]byte(txout.Address), byte(txout.Amount))
  hash := sha256.Sum256(txoutBytes)
  return string(hash[:]);
}

type TXIN struct {
  Sign     string
  IndexRef int
  IdRef    string
  //...
}

func (txin TXIN) Hash() string {
  txinBytes := txin.Sign + string(txin.IndexRef) + txin.IdRef
  hash := sha256.Sum256([]byte(txinBytes))
  return string(hash[:]);
}

type Transaction struct {
  Id    string
  Txin  []TXIN
  Txout []TXOUT
}

func (tx Transaction) Hash() string {
  var hash [32]byte
  for _, txin := range tx.Txin {
      txinHash := txin.Hash()
      hash = sha256.Sum256(append(hash[:], txinHash[:]...))
  }

  for _, txout := range tx.Txout {
      txoutHash := txout.Hash()
      hash = sha256.Sum256(append(hash[:], txoutHash[:]...))
  }

  return string(hash[:]);
}

type Block struct {
  Timestamp string
  Hash      string
  PrevHash  string
  Txs []Transaction
}

var Blockchain []Block
var PendingTxs []Transaction

var lastBlock Block

var pubKey []byte

func (block Block) HashBlock() string {
  var hash [32]byte
  for _, tx := range block.Txs {
    txHash := tx.Hash()
    hash = sha256.Sum256(append(hash[:], txHash[:]...))
  }

  return string(hash[:])
}

func (block Block) CountMyMoney() int {
  money := 0;
  // fmt.Println("count...");
  for _, tx := range block.Txs {
    // fmt.Println("test1");
    for _, txout := range tx.Txout {
      // fmt.Println("test2", base64.StdEncoding.EncodeToString(txout.Address),  " pub = ", base64.StdEncoding.EncodeToString(pubKey));
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        money += txout.Amount;
      }
    }
  }

  return money
}

// func generateBlock(oldBlock Block, BPM int) (Block, error) {
//   var newBlock Block
//
//   t := time.Now()
//
//   newBlock.Index = oldBlock.Index + 1
//   newBlock.Timestamp = t.String()
//   //newBlock.BPM = BPM
//   newBlock.PrevHash = oldBlock.Hash
//   newBlock.Hash = hashBlock(newBlock)
//
//   return newBlock, nil
// }

// func isBlockValid(newBlock, oldBlock Block) bool {
//   if oldBlock.Index + 1 != newBlock.Index {
//     return false
//   }
//   if oldBlock.Hash != newBlock.PrevHash {
//     return false
//   }
//   if hashBlock(newBlock) != newBlock.Hash {
//     return false
//   }
//
//   return false
// }

// func replaceChain(newBlocks []Block) {
//   if len(newBlocks) > len(Blockchain) {
//     Blockchain = newBlocks
//   }
// }

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
  // muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
  // muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
  return muxRouter
}

// func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
//   bytes, err := json.MarshalIndent(Blockchain, "", " ")
//   if err != nil {
//     http.Error(w, err.Error(), http.StatusInternalServerError)
//     return
//   }
//
//   io.WriteString(w, string(bytes))
// }
//
// type Message struct {
//   BPM int
// }

// func handleWriteBlock(w http.ResponseWriter, r *http.Request) {
//   var m Message
//
//   decoder := json.NewDecoder(r.Body)
//   if err := decoder.Decode(&m); err != nil {
//     respondWithJSON(w, r, http.StatusBadRequest, r.Body)
//     return
//   }
//
//   defer r.Body.Close()
//
//   newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
//   if (err != nil) {
//     respondWithJSON(w, r, http.StatusInternalServerError, m)
//     return
//   }
//
//   if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {
//     newBlockchain := append(Blockchain, newBlock)
//     replaceChain(newBlockchain)
//     spew.Dump(Blockchain)
//   }
//
//   respondWithJSON(w, r, http.StatusCreated, newBlock)
// }
//
// func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {
//   response, err := json.MarshalIndent(payload, "", " ")
//   if err != nil {
//     w.WriteHeader(http.StatusInternalServerError)
//     w.Write([]byte("HTTP 500: Internal Server Error"))
//   }
//   w.WriteHeader(code)
//   w.Write(response)
// }

func loadFiles(blockfile string, wallet string) {
  var bfile, _ = os.OpenFile(blockfile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer bfile.Close()
  reader := bufio.NewReader(bfile)
  line, _, _ := reader.ReadLine()

  if string(line) != "" {
    log.Print(string(line))
    var err = json.Unmarshal(line, &lastBlock)
    if err != nil {
      log.Fatal(err)
    }
  }

  //
  // randReader := rand.Reader
	// bitSize := 256
  //
	// key, err := rsa.GenerateKey(randReader, bitSize)
  //
  // spew.Dump(key)

  file, _ := ioutil.ReadFile(wallet)

  if string(file) == "" {
    return
  }
  // line, _, _ = reader.Read()
  // log.Print(string(file))
  // var private ecdsa.PrivateKey
  private, err := x509.ParseECPrivateKey(file)
  if err != nil {
    log.Fatal(err)
    return
  }
  pubKey = append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
  h := base64.StdEncoding.EncodeToString(pubKey)
  log.Print("Load: ", h)

  // if err != nil {
  //   log.Fatal(err, " err2")
  // }
}

type FriendlyTxInfo struct {
  Confirmed bool
  Type      string
  Amount    int
}

func IsMySpending (tx Transaction) bool {
  if len(tx.Txin) == 0 {
    return false
  }

  txin := tx.Txin[0]

  curve := elliptic.P256()
  sigLen := len(txin.Sign)

  r := big.Int{}
  s := big.Int{}
  r.SetBytes([]byte(txin.Sign)[:(sigLen / 2)])
  s.SetBytes([]byte(txin.Sign)[(sigLen / 2):])

  x := big.Int{}
  y := big.Int{}
  keyLen := len(pubKey)
  x.SetBytes(pubKey[:(keyLen / 2)])
  y.SetBytes(pubKey[(keyLen / 2):])

  rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

  isMySpending := ecdsa.Verify(&rawPubKey, []byte(txin.IdRef), &r, &s)

  if isMySpending {
    return true
  }
  return false
}

func showTransactionsWithStatus (txs []Transaction, status string) {
  for _, tx := range txs {
    isMy := IsMySpending(tx)
    txSpendings := 0
    for _, txout := range tx.Txout {
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        if isMy {
          fmt.Println("change                 ", txout.Amount, "    ", status);
        } else {
          if tx.Txin[0].Sign == "" {
            fmt.Println("income (coinbase)      ", txout.Amount, "    ", status);
          } else {
            fmt.Println("income                 ", txout.Amount, "    ", status);
          }
        }
      }

      txSpendings += txout.Amount
    }

    if isMy {
      fmt.Println("outcome                ", txSpendings, "    ", status);
    }
  }
}

func showTransactions () {
  var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()
  reader := bufio.NewReader(blockfile)

  for {
    line, _, err := reader.ReadLine()
    if len(line) == 0 {
      break

    }
    if err != nil {
      log.Fatal(err)
      return
    }

    var block Block
    err = json.Unmarshal(line, &block)
    if err != nil {
      log.Fatal(err)
      return
    }

    // balance += block.CountMyMoney()
    //
    // spent := make(map[string]byte)
    showTransactionsWithStatus(block.Txs, "confirmed")
  }
  // fmt.Println("trying...")
  showTransactionsWithStatus(PendingTxs, "pending")
}

func getUnspentTxs(limit int) ([]Transaction, []int) {
  var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()
  reader := bufio.NewReader(blockfile)

  var unspentTxs []Transaction
  var txIndexes  []int
  lineIndex := -1
  balance   := 0
  for {
    line, _, err := reader.ReadLine()
    if len(line) == 0 {
      break
    }
    if err != nil {
      log.Fatal(err)
      return unspentTxs, txIndexes
    }

    var block Block
    err = json.Unmarshal(line, &block)
    if err != nil {
      log.Fatal(err)
      return unspentTxs, txIndexes
    }

    // balance += block.CountMyMoney()

    // money := -1
    spent := make(map[string]int)

    for _, tx := range PendingTxs {
      if IsMySpending(tx) {
        spent[tx.Txin[0].IdRef] = -1
      }
    }

    for _, tx := range block.Txs {
      lineIndex+=1
      if  _, ok := spent[tx.Id]; ok {
        continue
      }
      for _, txout := range tx.Txout {
        /// assume there is only one out tx per wallet
        if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
          unspentTxs = append(unspentTxs, tx)
          txIndexes  = append(txIndexes, lineIndex)
          balance   += txout.Amount
          // break
        }
      }

      if limit > 0 && balance >= limit {
        return unspentTxs, txIndexes
      }

      r := big.Int{}
      s := big.Int{}
      for _, txin := range tx.Txin {
        if len(txin.Sign) == 0 {
          continue
        }

        curve := elliptic.P256()

		    sigLen := len(txin.Sign)
		    r.SetBytes([]byte(txin.Sign)[:(sigLen / 2)])
		    s.SetBytes([]byte(txin.Sign)[(sigLen / 2):])

        x := big.Int{}
		    y := big.Int{}
		    keyLen := len(pubKey)
		    x.SetBytes(pubKey[:(keyLen / 2)])
		    y.SetBytes(pubKey[(keyLen / 2):])

        rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

        isMySpending := ecdsa.Verify(&rawPubKey, []byte(txin.IdRef), &r, &s)

        if isMySpending {
          spent[txin.IdRef] = 1
        }
      }

    }
    // return money
  }

  if (balance < limit) {
    return []Transaction{}, []int{}
  }

  return unspentTxs, txIndexes
}

func getBalance () (int, int) {
  unspentTxs, _ := getUnspentTxs(-1)
  pendingTxsOut, pendingTxsIn := getPendingTransactions()
  pendingTxsInMap  := make(map[string]TXIN)

  for _, t := range pendingTxsIn {
    pendingTxsInMap[t.IdRef] = t
  }

  ConfirmedBalance := 0
  for _, tx := range unspentTxs {
    if _, ok := pendingTxsInMap[tx.Id]; ok {
      continue
    }

    for _, txout := range tx.Txout {
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        ConfirmedBalance += txout.Amount
      }
    }
  }

  UnconfirmedBalance := 0
  for _, tx := range pendingTxsOut {
    UnconfirmedBalance += tx.Amount
  }

  return ConfirmedBalance, UnconfirmedBalance
}

func getPrivateKey() ecdsa.PrivateKey {
  file, _ := ioutil.ReadFile("wallet.dat")

  if string(file) == "" {
    return ecdsa.PrivateKey{}
  }
  // line, _, _ = reader.Read()
  // log.Print(string(file))
  private, err := x509.ParseECPrivateKey(file)
  if err != nil {
    log.Fatal(err)
    return ecdsa.PrivateKey{}
  }
  copy := *private
  return copy
}

func getPendingTransactions() ([]TXOUT, []TXIN) {
  var txsout []TXOUT
  var txsin  []TXIN
  for _, tx := range PendingTxs {
    for _, txout := range tx.Txout {
      // fmt.Print(txout.Address)

      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        txsout = append(txsout, txout)
      }
    }
    for _, txin  := range tx.Txin {
      r := big.Int{}
      s := big.Int{}
      if len(txin.Sign) == 0 {
        continue
      }

      curve := elliptic.P256()

      sigLen := len(txin.Sign)
      r.SetBytes([]byte(txin.Sign)[:(sigLen / 2)])
      s.SetBytes([]byte(txin.Sign)[(sigLen / 2):])

      x := big.Int{}
      y := big.Int{}
      keyLen := len(pubKey)
      x.SetBytes(pubKey[:(keyLen / 2)])
      y.SetBytes(pubKey[(keyLen / 2):])

      rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

      isMySpending := ecdsa.Verify(&rawPubKey, []byte(txin.IdRef), &r, &s)

      if isMySpending {
        txsin = append(txsin, txin)
      }
    }
  }
  return txsout, txsin
}

func CreateTransaction(unspentTxs []Transaction, txIndexes []int, amount int, address string) Transaction {
  var txsin []TXIN
  privateKey := getPrivateKey()

  size := privateKey.Curve
  fmt.Println(size)
  // spew.Dump(*(&privateKey))

  totalInput := 0
  for i, tx  := range unspentTxs {
    r,s, _   := ecdsa.Sign(rand.Reader, &privateKey, []byte(tx.Id))
    sign     := append(r.Bytes(),s.Bytes()...)
    txin     := TXIN{string(sign), txIndexes[i], tx.Id}
    txsin     = append(txsin, txin)
    for _, txout := range tx.Txout {
      totalInput += txout.Amount
    }
  }

  var txsout []TXOUT
  txout  := TXOUT{address, amount}
  change := TXOUT{base64.StdEncoding.EncodeToString(pubKey), totalInput - amount}
  txsout = append(txsout, txout)
  if change.Amount > 0 {
    txsout = append(txsout, change)
  }

  newtx   := Transaction{"", txsin, txsout}
  spew.Dump(newtx)
  newtx.Id = newtx.Hash()
  return newtx
}

func send() {
  var amount  int
  var address string


  fmt.Print("Amount: ")
  _, err := fmt.Scanf("%d", &amount)

  if err != nil {
    fmt.Println(err)
  }

  fmt.Print("Address: ")
  _, _ = fmt.Scanf("%s", &address)

  unspentTxs, txIndexes := getUnspentTxs(amount)
  if len(unspentTxs) == 0 {
    fmt.Println("Not enough money")
    return
  }

  sendTx := CreateTransaction(unspentTxs, txIndexes, amount, address)

  PendingTxs = append(PendingTxs, sendTx)
  // txsToSpend = getSufficientInput(amount)

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

  loadFiles("blockchain.dat", "wallet.dat")

  go func() {
    log.Print("len = ", len(lastBlock.Hash))
    if len(lastBlock.Hash) == 0 {
      var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer blockfile.Close()
      /// Hardcode genesis block
      txin  := TXIN{"", 0, ""}
      txout := TXOUT{"zZvNvCqtvZ3FhUjO+QjoiBQoj+Pgj5GNJDO7z2HifSxGvDfjKHuutUQWLCHifFyXfYNss/LAxYschi3oLLnKww==", 50}
      tx    := Transaction{"", []TXIN{txin}, []TXOUT{txout}}
      tx.Id  = tx.Hash()
      txs   := []Transaction{tx}
      genesisBlock := Block{"10.03.2018 easy peasy lemon squeezy", "", "GENESIS", txs}
      genesisBlock.Hash = genesisBlock.HashBlock()
      spew.Dump(genesisBlock)
      Blockchain = append(Blockchain, genesisBlock)
      str, err2 := json.Marshal(genesisBlock)
      if err2 != nil {
        log.Fatal(err, "err1")
        return
      }
      blockfile.WriteString(string(str) + "\n")
    }

    if len(pubKey)==0 {
      // randReader := rand.Reader
    	// bitSize := 256

      curve := elliptic.P256()
    	private, err := ecdsa.GenerateKey(curve, rand.Reader)
    	if err != nil {
    		log.Panic(err)
    	}

      spew.Dump(private)

      var wallet, _    = os.OpenFile("wallet.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer wallet.Close()

    	pubKey = append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

    	// key, _ := rsa.GenerateKey(randReader, bitSize)
      // pub     = key.PublicKey
      str, _ := x509.MarshalECPrivateKey(private)
      h := base64.StdEncoding.EncodeToString(pubKey)
      log.Print("GENERATE ", h)
      wallet.Write(str)
    }

  } ()

  go func() {
    getInput()
  } ()

  log.Fatal(run())
}
