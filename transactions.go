package main

import (
  "crypto/sha256"
  "crypto/rand"
  "crypto/ecdsa"
  "encoding/base64"
  "fmt"
  "github.com/davecgh/go-spew/spew"
  "sync"
)

var PendingTransactionsMutex sync.Mutex

type TXOUT struct {
  Address string;
  Amount int;
}

func (txout TXOUT) Hash() []byte {
  txoutBytes := append([]byte(txout.Address), byte(txout.Amount))
  hash := sha256.Sum256(txoutBytes)
  return hash[:];
}

type TXIN struct {
  Sign     []byte
  IdRef    []byte
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

func CreateTransaction(unspentTxs []Transaction, amount int, address string) Transaction {
  var txsin []TXIN
  privateKey := getPrivateKey()

  totalInput := 0
  for _, tx  := range unspentTxs {
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

func showTransactions () {
  IterateBlockchainForward(func(block Block) (bool, error) {
    showTransactionsWithStatus(block.Txs, "confirmed")
    return false, nil
  })

  PendingTransactionsMutex.Lock()
  defer PendingTransactionsMutex.Unlock()
  showTransactionsWithStatus(PendingTxs, "pending")
}


func showTransactionsWithStatus (txs []Transaction, status string) {
  for _, tx := range txs {
    isMy := IsMySpending(tx)
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
