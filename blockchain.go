package main

import (
  "crypto/sha256"
  // "crypto/rsa"
  // "crypto/rand"
  // "crypto/elliptic"
  // "crypto/ecdsa"
  // "crypto/x509"
  // "encoding/base64"
  // "encoding/hex"
  // "encoding/json"
  // "io"
  // "log"
  // "net/http"
  // "os"
  // "time"
  // "bufio"
  // "io/ioutil"
  // "math/big"
  // "fmt"
  // "bytes"
  // "github.com/davecgh/go-spew/spew"
  // "github.com/gorilla/mux"
  // "github.com/joho/godotenv"
  // "github.com/serverhorror/rog-go/reverse"
  // "github.com/dustin/go-hashset"
)

type Block struct {
  Timestamp string
  Hash      []byte
  PrevHash  []byte
  Txs []Transaction
  Nonce     []byte
}

var blockchainFileName = "blockchain.dat"

func (block Block) HashBlock() []byte {
  var hash [32]byte
  for _, tx := range block.Txs {
    txHash := tx.Hash()
    hash = sha256.Sum256(append(hash[:], txHash[:]...))
    hash = sha256.Sum256(append(hash[:], []byte(block.Nonce)...))
  }

  return hash[:]
}
