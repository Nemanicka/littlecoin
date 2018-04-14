package main

import (
  "crypto/sha256"
  "crypto/rand"
  "encoding/base64"
  "time"
  "fmt"
  "github.com/davecgh/go-spew/spew"
  "errors"
)

var mining bool
var VerifiedPendingTxsHash []byte


func Mine() {
  for {
    var nonce [32]byte
    _, err := rand.Read(nonce[:])

    if err != nil {
      continue
    }

    PendingTransactionsMutex.Lock()
    lastBlock, _ := getLastBlock()

    timestamp  := time.Now().String()
    blockBytes := append([]byte(timestamp), lastBlock.Hash...)
    blockBytes  = append(blockBytes, nonce[:]...)
    blockBytes  = append(blockBytes, VerifiedPendingTxsHash...)

    hash := sha256.Sum256(blockBytes)
    if hash[0] == 0 && hash[1] == 0 && hash[2] == 0 {
      newBlock := Block{timestamp, hash[:], lastBlock.Hash, VerifiedPendingTxs, nonce[:]}
      spew.Dump(newBlock)
      AppendToBlockChain(newBlock)
      VerifiedPendingTxs = []Transaction{}
      PendingTxs         = []Transaction{}
      propagateBlock(newBlock)
    }

    PendingTransactionsMutex.Unlock()

    if !mining {
      fmt.Println("Stopped mining")
      fmt.Print(">")
      break
    }
  }
}

func OnPendingTxsAdded(sendTx Transaction) {
  PendingTransactionsMutex.Lock()
  defer PendingTransactionsMutex.Unlock()

  PendingTxs = append(PendingTxs, sendTx)

  if !mining {
    return
  }

  txIdsMap := make(map[string]bool)
  sendingAmount := 0
  investedAmount := 0

  for _, txin := range sendTx.Txin {
    txIdsMap[string(txin.IdRef)] = true
  }

  for _, txout := range sendTx.Txout {
    sendingAmount += txout.Amount
  }

  IterateBlockchainBackward(func(block Block) (bool, error)  {
    for _, tx := range block.Txs {
      for _, txin := range tx.Txin {
        if _, ok := txIdsMap[string(txin.IdRef)]; ok {
          fmt.Print("Transaction ", string(txin.IdRef), " is already used");
          return true, errors.New("Already user tx")
        }
      }

      // if this is a transaction refereced in sendTx, count money
      if _, ok := txIdsMap[string(tx.Id)]; ok {
        for _, txout := range tx.Txout {
          address, _ := base64.StdEncoding.DecodeString(txout.Address)
          if DoesKeyUnlocksTransaction(address, sendTx.Txin[0]) {
            investedAmount += txout.Amount
          }
        }
      }
    }
    return false, nil
  })

  if sendingAmount > investedAmount {
    fmt.Println("Not enough money in tx ", sendTx.Id)
  }

  var VerifiedPendingTxsHash  [32]byte
  VerifiedPendingTxs = append(VerifiedPendingTxs, sendTx);
  for _, tx := range VerifiedPendingTxs {
    bytes := append(VerifiedPendingTxsHash[:], []byte(tx.Id)...)
    VerifiedPendingTxsHash = sha256.Sum256(bytes)
  }
}
