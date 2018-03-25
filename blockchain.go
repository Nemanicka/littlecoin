package main

import (
  "crypto/sha256"
  "encoding/json"
  "os"
  "bufio"
  "errors"
  "github.com/serverhorror/rog-go/reverse"
)

var blockchainFileName = "blockchain.dat"

type Block struct {
  Timestamp string
  Hash      []byte
  PrevHash  []byte
  Txs []Transaction
  Nonce     []byte
}

func (block Block) HashBlock() []byte {
  var hash [32]byte
  for _, tx := range block.Txs {
    txHash := tx.Hash()
    hash = sha256.Sum256(append(hash[:], txHash[:]...))
    hash = sha256.Sum256(append(hash[:], []byte(block.Nonce)...))
  }

  return hash[:]
}

func CreateGenesisBlock() Block {
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
  return genesisBlock
}



func AppendToBlockChain(block Block) error {
  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()

  bytes, err := json.Marshal(block)
  if err != nil {
    return errors.New("Failed to append to block chain")
  }

  blockfile.WriteString(string(bytes) + "\n")
  return nil
}

//
// type BlockchainReader struct {
//   reader *bufio.Reader
//   rReader *reverse.Scanner
//   file   *os.File
// }
//
// func CreateForward() error {
//   b := BlockchainReader{}
//   b.file, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
//   b.reader = bufio.NewReader(b.file)
//   return nil
// }
//
// func (b *BlockchainReader) Destroy() error {
//     b.file.Close()
//     return nil
// }
//
// func (b *BlockchainReader) NextBlock() (Block, error) {
//   line, _, err := b.reader.ReadLine()
//   if len(line) == 0 {
//     return Block{}, nil
//   }
//
//   if err != nil {
//     return Block{}, err
//   }
//
//   var block Block
//   err = json.Unmarshal(line, &block)
//   if err != nil {
//     return Block{}, err
//   }
//
//   return block, nil
// }
//

type onBlock func(block Block) (bool, error)

func IterateBlockchainForward(lambda onBlock) error {
  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()
  reader := bufio.NewReader(blockfile)

  for {
    line, _, err := reader.ReadLine()
    if len(line) == 0 {
      return nil
    }

    if err != nil {
      return err
    }

    var block Block
    err = json.Unmarshal(line, &block)
    if err != nil {
      return err
    }

    stop := false
    stop, err = lambda(block)
    if err != nil {
      return err
    }

    if stop {
      return nil
    }
  }

  return nil
}

func IterateBlockchainBackward(lambda onBlock) error {
  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()

  scanner := reverse.NewScanner(blockfile)
  scanner.Split(bufio.ScanLines)

  for {
    retcode := scanner.Scan()
    if !retcode {
      break
    }

    line := scanner.Text()

    var block Block
    err := json.Unmarshal([]byte(line), &block)
    if err != nil {
      return err
    }

    stop := false
    stop, err = lambda(block)
    if err != nil {
      return err
    }

    if stop {
      return nil
    }
  }

  return nil
}
