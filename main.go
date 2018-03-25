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
  "github.com/serverhorror/rog-go/reverse"
  // "github.com/dustin/go-hashset"
)

type TXOUT struct {
  Address string;
  Amount int;
  //...
}
//
func (txout TXOUT) Hash() []byte {
  txoutBytes := append([]byte(txout.Address), byte(txout.Amount))
  hash := sha256.Sum256(txoutBytes)
  return hash[:];
}

type TXIN struct {
  Sign     []byte
  IdRef    []byte
  //...
}

func (txin TXIN) Hash() []byte {
  txinBytes := append(txin.Sign, txin.IdRef...)
  hash := sha256.Sum256(txinBytes)
  return hash[:];
}

type Transaction struct {
  Id    []byte
  Txin  []TXIN
  Txout []TXOUT
}

func (tx Transaction) Hash() []byte {
  var hash [32]byte
  for _, txin := range tx.Txin {
      txinHash := txin.Hash()
      hash = sha256.Sum256(append(hash[:], txinHash[:]...))
  }

  for _, txout := range tx.Txout {
      txoutHash := txout.Hash()
      hash = sha256.Sum256(append(hash[:], txoutHash[:]...))
  }

  return hash[:];
}

type Block struct {
  Timestamp string
  Hash      []byte
  PrevHash  []byte
  Txs []Transaction
  Nonce     []byte
}

var Blockchain []Block
var PendingTxs []Transaction
var VerifiedPendingTxs []Transaction
var mining bool

var lastBlock Block
var VerifiedPendingTxsHash []byte

var pubKey []byte

func (block Block) HashBlock() []byte {
  var hash [32]byte
  for _, tx := range block.Txs {
    txHash := tx.Hash()
    hash = sha256.Sum256(append(hash[:], txHash[:]...))
    hash = sha256.Sum256(append(hash[:], []byte(block.Nonce)...))
  }

  return hash[:]
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

func propagateBlock(block Block) {

}

func makeMuxRouter() http.Handler {
  muxRouter := mux.NewRouter()
  // muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
  // muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")
  return muxRouter
}
//
// Timestamp string
// Hash      string
// PrevHash  string
// Txs []Transaction
// Nonce     string


func Mine() {

  for {
    var nonce [32]byte
    _, err := rand.Read(nonce[:])

    if err != nil {
      continue
    }

    // newBlock           = Block{}
    timestamp := time.Now().String()
    // prevHash  = lastBlock.Hash
    // newBlock.Txs       = VerifiedPendingTxs
    // newBlock.Nonce     = string(nonce)
    blockBytes := append([]byte(timestamp), lastBlock.Hash...)
    blockBytes  = append(blockBytes, nonce[:]...)
    blockBytes  = append(blockBytes, VerifiedPendingTxsHash...)

    hash := sha256.Sum256(blockBytes)
    if hash[0] == 0 && hash[1] == 0 && hash[2] == 0 {
      newBlock := Block{timestamp, hash[:], lastBlock.Hash, VerifiedPendingTxs, nonce[:]}
      // lastBlock = newBlock

      var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer blockfile.Close()
      /// Hardcode genesis block

      spew.Dump(newBlock)
      // Blockchain = append(Blockchain, lastBlock)
      str, err2 := json.Marshal(newBlock)
      if err2 != nil {
        fmt.Print(err)
        return
      }
      blockfile.WriteString(string(str) + "\n")
      lastBlock = newBlock
      VerifiedPendingTxs = []Transaction{}
      PendingTxs         = []Transaction{}
      propagateBlock(newBlock)
    }

    if !mining {
      fmt.Println("Stopped mining")
      fmt.Print(">")
      return
    }
  }

  if mining {
    Mine()
  } else {
    fmt.Println("Stopped mining")
    fmt.Print(">")
  }

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
  if _, err := os.Stat(blockfile); err == nil {
    var bfile, _ = os.OpenFile(blockfile, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
    defer bfile.Close()

    scanner := reverse.NewScanner(bfile)
    scanner.Split(bufio.ScanLines)
    scanner.Scan()
    line := scanner.Text()


    if line != "" {
      log.Print(line)
      var err = json.Unmarshal([]byte(line), &lastBlock)
      if err != nil {
        log.Fatal(err)
      }
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

}

type FriendlyTxInfo struct {
  Confirmed bool
  Type      string
  Amount    int
}

func keyUnlocksTransaction (key []byte, txin TXIN) bool {
  if len(txin.Sign) == 0 && len(txin.IdRef) == 0 {
    return false
  }

  curve := elliptic.P256()
  sigLen := len(txin.Sign)

  // fmt.Println("Sign = ", base64.StdEncoding.EncodeToString([]byte(txin.Sign)))

  r := big.Int{}
  s := big.Int{}
  r.SetBytes([]byte(txin.Sign)[:(sigLen / 2)])
  s.SetBytes([]byte(txin.Sign)[(sigLen / 2):])

  // fmt.Println("check r, s = ", r,s)

  x := big.Int{}
  y := big.Int{}
  keyLen := len(key)
  x.SetBytes(key[:(keyLen / 2)])
  y.SetBytes(key[(keyLen / 2):])

  rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

  isMySpending := ecdsa.Verify(&rawPubKey, []byte(txin.IdRef), &r, &s)

  if isMySpending {
    // fmt.Println("MY SPENT");
    return true
  }
  return false
}

func IsMySpending (tx Transaction) bool {
  if len(tx.Txin) == 0 {
    return false
  }

  txin := tx.Txin[0]

  return keyUnlocksTransaction(pubKey, txin)
}

func showTransactionsWithStatus (txs []Transaction, status string) {
  for _, tx := range txs {
    isMy := IsMySpending(tx)
    // txSpendings := 0
    outMap := make(map[string]int)
    for _, txout := range tx.Txout {
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        if isMy {
          fmt.Println("change                 ", txout.Amount, "    ", status);
        } else {
          if len(tx.Txin[0].Sign) == 0 {
            fmt.Println("income (coinbase)      ", txout.Amount, "    ", status);
          } else {
            fmt.Println("income                 ", txout.Amount, "    ", status);
          }
        }
      } else {
        if isMy {
          outMap[txout.Address] += txout.Amount
        }
      }
    }

    if isMy {
      for k, v := range outMap {
        fmt.Println("outcome                ", v, "     ", status, "   ", k);
      }
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
      fmt.Println(err)
      return
    }

    var block Block
    err = json.Unmarshal(line, &block)
    if err != nil {
      fmt.Println(err)
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

func getUnspentTxs(limit int) ([]Transaction) {
  var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()

  var unspentTxs []Transaction
  balance   := 0

  scanner := reverse.NewScanner(blockfile)
  scanner.Split(bufio.ScanLines)
  spent := make(map[string]int)

  for {
    retcode := scanner.Scan()
    if !retcode {
      break
    }

    line := scanner.Text()

    var block Block
    err := json.Unmarshal([]byte(line), &block)
    if err != nil {
      fmt.Println(err)
      return unspentTxs
    }

    // balance += block.CountMyMoney()

    // money := -1

    for _, tx := range PendingTxs {
      if IsMySpending(tx) {
        spent[string(tx.Txin[0].IdRef)] = 1
      }
    }

    for _, tx := range block.Txs {
      fmt.Println("Hashmap len = ", len(spent))

      for k , _ := range spent {
        fmt.Println("spent map = ", k)
      }

      if IsMySpending(tx) {
        fmt.Println("spent id = ", tx.Txin[0].IdRef)
        spent[string(tx.Txin[0].IdRef)] = 1
      }

      if  _, ok := spent[string(tx.Id)]; ok {
        fmt.Println("Cont?")
        continue
      } else {
        fmt.Println("no")
      }

      isUnspent := false
      for _, txout := range tx.Txout {
        /// assume there is only one out tx per wallet
        if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
          // txIndexes  = append(txIndexes, lineIndex)
          balance   += txout.Amount
          isUnspent = true
        }
      }

      if isUnspent {
        fmt.Println("unspent = ", tx.Id)
        unspentTxs = append(unspentTxs, tx)
      }

      if limit > 0 && balance >= limit {
        return unspentTxs
      }
    }
  }


  if (balance < limit) {
    return []Transaction{}
  }

  return unspentTxs
}

func getBalance () (int, int) {
  unspentTxs := getUnspentTxs(-1)
  pendingTxsOut, pendingTxsIn := getPendingTransactions()
  pendingTxsInMap  := make(map[string]TXIN)

  for _, t := range pendingTxsIn {
    pendingTxsInMap[string(t.IdRef)] = t
  }

  ConfirmedBalance := 0
  for _, tx := range unspentTxs {
    if _, ok := pendingTxsInMap[string(tx.Id)]; ok {
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
    fmt.Println(err)
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

func OnPendingTxsAdded(sendTx Transaction) {
  if !mining {
    return
  }

  var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()
  txIdsMap := make(map[string]bool)

  for _, txin := range sendTx.Txin {
    txIdsMap[string(txin.IdRef)] = true
  }

  // Check that sendTx references valid tx, and there is enough money
  sendingAmount := 0
  investedAmount := 0
  for _, txout := range sendTx.Txout {
    sendingAmount += txout.Amount
  }

  fmt.Println("sendingAmount = ", sendingAmount)

  scanner := reverse.NewScanner(blockfile)
  scanner.Split(bufio.ScanLines)

  // Check input txs are not already spent
  for {
    retcode := scanner.Scan()
    if !retcode {
      break
    }

    line := scanner.Text()

    var block Block
    err := json.Unmarshal([]byte(line), &block)
    if err != nil {
      fmt.Println(err)
    }

    for _, tx := range block.Txs {
      for _, txin := range tx.Txin {
        if _, ok := txIdsMap[string(txin.IdRef)]; ok {
          fmt.Print("Transaction ", string(txin.IdRef), " is already used");
          return
        }
      }

      // if this is a transaction refereced in sendTx, count money
      if _, ok := txIdsMap[string(tx.Id)]; ok {
        for _, txout := range tx.Txout {
          address, _ := base64.StdEncoding.DecodeString(txout.Address)
          if keyUnlocksTransaction(address, sendTx.Txin[0]) {
            investedAmount += txout.Amount
          }
        }
      }

      fmt.Println("investedAmount = ", investedAmount)

    }
  }

  if sendingAmount > investedAmount {
    fmt.Println("Not enough money in tx ", sendTx.Id)
  }

  {
    var VerifiedPendingTxsHash  [32]byte
    VerifiedPendingTxs = append(VerifiedPendingTxs, sendTx);
    for _, tx := range VerifiedPendingTxs {
      bytes := append(VerifiedPendingTxsHash[:], []byte(tx.Id)...)
      VerifiedPendingTxsHash = sha256.Sum256(bytes)
    }
  }
}

func CreateTransaction(unspentTxs []Transaction, amount int, address string) Transaction {
  var txsin []TXIN
  privateKey := getPrivateKey()

  size := privateKey.Curve
  fmt.Println(size)
  // spew.Dump(*(&privateKey))

  totalInput := 0
  for _, tx  := range unspentTxs {
    fmt.Println("unspent id = ", tx.Id)

    r,s, _   := ecdsa.Sign(rand.Reader, &privateKey, []byte(tx.Id))
    sign     := append(r.Bytes(),s.Bytes()...)
    txin     := TXIN{sign, tx.Id}

    txsin     = append(txsin, txin)
    for _, txout := range tx.Txout {
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        totalInput += txout.Amount
      }
    }
  }

  var txsout []TXOUT
  txout  := TXOUT{address, amount}
  change := TXOUT{base64.StdEncoding.EncodeToString(pubKey), totalInput - amount}
  txsout = append(txsout, txout)
  if change.Amount > 0 {
    txsout = append(txsout, change)
  }

  newtx   := Transaction{[]byte{}, txsin, txsout}
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

  unspentTxs:= getUnspentTxs(amount)
  if len(unspentTxs) == 0 {
    fmt.Println("Not enough money")
    return
  }

  sendTx := CreateTransaction(unspentTxs, amount, address)

  PendingTxs = append(PendingTxs, sendTx)
  OnPendingTxsAdded(sendTx);
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

  loadFiles("blockchain.dat", "wallet.dat")

  go func() {
    log.Print("len = ", len(lastBlock.Hash))
    if len(lastBlock.Hash) == 0 {
      var blockfile, _ = os.OpenFile("blockchain.dat", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer blockfile.Close()
      /// Hardcode genesis block
      txin  := TXIN{[]byte{}, []byte{}}
      txout := TXOUT{"zZvNvCqtvZ3FhUjO+QjoiBQoj+Pgj5GNJDO7z2HifSxGvDfjKHuutUQWLCHifFyXfYNss/LAxYschi3oLLnKww==", 50}
      tx    := Transaction{[]byte{}, []TXIN{txin}, []TXOUT{txout}}
      tx.Id  = tx.Hash()
      txs   := []Transaction{tx}
      genesisBlock := Block{"10.03.2018 easy peasy lemon squeezy", []byte{}, []byte{'G'}, txs, []byte{}}
      genesisBlock.Hash = genesisBlock.HashBlock()
      spew.Dump(genesisBlock)
      Blockchain = append(Blockchain, genesisBlock)
      str, err2 := json.Marshal(genesisBlock)
      fmt.Println("genesis str = ", string(str))
      var xBlock Block
      json.Unmarshal([]byte(str), &xBlock)

      fmt.Println("------------------")
      spew.Dump(xBlock)
      if err2 != nil {
        log.Fatal(err)
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
