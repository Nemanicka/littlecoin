package main

import (
  "crypto/sha256"
  // "crypto/rsa"
  "crypto/rand"
  // "crypto/elliptic"
  // "crypto/ecdsa"
  // "crypto/x509"
  "encoding/base64"
  // "encoding/hex"
  "encoding/json"
  // "io"
  // "log"
  // "net/http"
  "os"
  "time"
  "bufio"
  // "io/ioutil"
  // "math/big"
  "fmt"
  // "bytes"
  "github.com/davecgh/go-spew/spew"
  // "github.com/gorilla/mux"
  // "github.com/joho/godotenv"
  "github.com/serverhorror/rog-go/reverse"
  // "github.com/dustin/go-hashset"
)

var mining bool
var lastBlock Block
var VerifiedPendingTxsHash []byte


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

      var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
      defer blockfile.Close()

      spew.Dump(newBlock)
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

func OnPendingTxsAdded(sendTx Transaction) {
  if !mining {
    return
  }

  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
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
