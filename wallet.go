package main

import (
  "crypto/rand"
  "crypto/elliptic"
  "crypto/ecdsa"
  "crypto/x509"
  "os"
  "io/ioutil"
  "fmt"
  "sync"
)

var pubKey []byte
var walletFileName = "wallet.dat"
var walletMutex sync.Mutex

func getPrivateKey() ecdsa.PrivateKey {
  walletMutex.Lock()
  defer walletMutex.Unlock()
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

func CreateWalllet() error {
  walletMutex.Lock()
  defer walletMutex.Unlock()

  curve := elliptic.P256()
  private, err := ecdsa.GenerateKey(curve, rand.Reader)
  if err != nil {
    return err
  }

  var wallet, _ = os.OpenFile(walletFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer wallet.Close()

  pubKey = append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
  keyStr, _ := x509.MarshalECPrivateKey(private)
  wallet.Write(keyStr)
  return nil
}
