package main

import (
  // "crypto/sha256"
  // "crypto/rsa"
  // "crypto/rand"
  // "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/x509"
  // "encoding/base64"
  // "encoding/hex"
  // "encoding/json"
  // "io"
  // "log"
  // "net/http"
  // "os"
  // "time"
  // "bufio"
  "io/ioutil"
  // "math/big"
  "fmt"
  // "bytes"
  // "github.com/davecgh/go-spew/spew"
  // "github.com/gorilla/mux"
  // "github.com/joho/godotenv"
  // "github.com/serverhorror/rog-go/reverse"
  // "github.com/dustin/go-hashset"
)

var pubKey []byte
var walletFileName = "wallet.dat"

func getPrivateKey() ecdsa.PrivateKey {
  file, _ := ioutil.ReadFile(walletFileName)

  if string(file) == "" {
    return ecdsa.PrivateKey{}
  }

  private, err := x509.ParseECPrivateKey(file)
  if err != nil {
    fmt.Println(err)
    return ecdsa.PrivateKey{}
  }

  copy := *private
  return copy
}
