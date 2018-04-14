package main

import (
  "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/x509"
  "encoding/base64"
  // "encoding/json"
  "log"
  "os"
  // "bufio"
  "io/ioutil"
  "math/big"
  "fmt"
  // "github.com/serverhorror/rog-go/reverse"
)


func propagateBlock(block Block) {

}

var PendingTxs []Transaction
var VerifiedPendingTxs []Transaction

func loadFiles() {
  loadAddresses()

  blockchainMutex.Lock()
  defer blockchainMutex.Unlock()

  if _, err := os.Stat(blockchainFileName); err == nil {
    var bfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
    defer bfile.Close()

    // scanner := reverse.NewScanner(bfile)
    // scanner.Split(bufio.ScanLines)
    // scanner.Scan()
    // line := scanner.Text()
    //
    // Block
    // if line != "" {
    //   log.Print(line)
    //   var err = json.Unmarshal([]byte(line), &lastBlock)
    //   if err != nil {
    //     log.Fatal(err)
    //   }
    // }
  }

  walletMutex.Lock()
  defer walletMutex.Unlock()

  file, _ := ioutil.ReadFile(walletFileName)

  if string(file) == "" {
    return
  }

  private, err := x509.ParseECPrivateKey(file)
  if err != nil {
    log.Fatal(err)
    return
  }

  pubKey = append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
  h := base64.StdEncoding.EncodeToString(pubKey)
  log.Print("Load: ", h)
}

func DoesKeyUnlocksTransaction(key []byte, txin TXIN) bool {
  if len(txin.Sign) == 0 && len(txin.IdRef) == 0 {
    return false
  }

  curve := elliptic.P256()
  sigLen := len(txin.Sign)

  r := big.Int{}
  s := big.Int{}
  r.SetBytes([]byte(txin.Sign)[:(sigLen / 2)])
  s.SetBytes([]byte(txin.Sign)[(sigLen / 2):])

  x := big.Int{}
  y := big.Int{}
  keyLen := len(key)
  x.SetBytes(key[:(keyLen / 2)])
  y.SetBytes(key[(keyLen / 2):])

  rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

  unlocks := ecdsa.Verify(&rawPubKey, []byte(txin.IdRef), &r, &s)

  if unlocks {
    return true
  }
  return false
}

func IsMySpending(tx Transaction) bool {
  if len(tx.Txin) == 0 {
    return false
  }

  txin := tx.Txin[0]

  return DoesKeyUnlocksTransaction(pubKey, txin)
}

func getUnspentTxs(limit int) ([]Transaction) {
  var unspentTxs []Transaction
  balance   := 0

  spent := make(map[string]int)

  // Store already spent transaction from pending txs
  PendingTransactionsMutex.Lock()
  for _, tx := range PendingTxs {
    if IsMySpending(tx) {
      spent[string(tx.Txin[0].IdRef)] = 1
    }
  }
  PendingTransactionsMutex.Unlock()

  IterateBlockchainBackward(func(block Block) (bool, error)  {
    for _, tx := range block.Txs {
      // Store already spent transaction from each block
      if IsMySpending(tx) {
        spent[string(tx.Txin[0].IdRef)] = 1
      }

      // If this txs is already spent, continue
      if  _, ok := spent[string(tx.Id)]; ok {
        continue
      }

      isUnspent := false
      for _, txout := range tx.Txout {
        if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
          balance   += txout.Amount
          isUnspent = true
        }
      }

      if isUnspent {
        unspentTxs = append(unspentTxs, tx)
      }

      if limit > 0 && balance >= limit {
        return true, nil
      }
    }
    return false, nil
  })


  if (balance < limit) {
    return []Transaction{}
  }

  return unspentTxs
}

func getBalance() (int, int) {
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

func getPendingTransactions() ([]TXOUT, []TXIN) {
  PendingTransactionsMutex.Lock()
  defer PendingTransactionsMutex.Unlock()

  var txsout []TXOUT
  var txsin  []TXIN
  for _, tx := range PendingTxs {
    for _, txout := range tx.Txout {
      if txout.Address == base64.StdEncoding.EncodeToString(pubKey) {
        txsout = append(txsout, txout)
      }
    }

    for _, txin  := range tx.Txin {
      if DoesKeyUnlocksTransaction(pubKey, txin) {
        txsin = append(txsin, txin)
      }
    }
  }
  return txsout, txsin
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

  OnPendingTxsAdded(sendTx);
}
