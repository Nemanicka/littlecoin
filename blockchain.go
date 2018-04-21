package main

import (
  "crypto/sha256"
  "encoding/json"
  "os"
  "bufio"
  "errors"
  "github.com/serverhorror/rog-go/reverse"
  "sync"
  "bytes"
)

var blockchainFileName = "blockchain.dat"
var blockchainMutex sync.Mutex

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

func deleteNLastBlocks(linesNum int) error {
  blockIndex := 0

  if (linesNum < 1) {
    return nil
  }

  blockchainMutex.Lock()
  defer blockchainMutex.Unlock()

  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR, 0644)
  defer blockfile.Close()

  scanner := reverse.NewScanner(blockfile)
  scanner.Split(bufio.ScanLines)

  truncateBytesNum := 0
  for {
    retcode := scanner.Scan()
    if !retcode {
      break
    }
    line := scanner.Text()
    truncateBytesNum += len(line)
    blockIndex += 1
  }

  os.Truncate(blockchainFileName, int64(truncateBytesNum))

  return nil
}

func AppendToBlockChain(block Block) error {
  blockchainMutex.Lock()
  defer blockchainMutex.Unlock()

  lastBlock, _ := getLastBlock()
  if len(lastBlock.Hash) != 0 && !bytes.Equal(lastBlock.Hash, block.PrevHash) {
    return errors.New("Previous hash is invalid")
  }

  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()

  bytes, err := json.Marshal(block)
  if err != nil {
    return errors.New("Failed to append to block chain")
  }

  blockfile.WriteString(string(bytes) + "\n")
  return nil
}

type onBlock func(block Block) (bool, error)

func IterateBlockchainForward(lambda onBlock) error {
  blockchainMutex.Lock()
  defer blockchainMutex.Unlock()

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
  blockchainMutex.Lock()
  defer blockchainMutex.Unlock()

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

func getLastBlock() (Block, error) {
  var blockfile, _ = os.OpenFile(blockchainFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
  defer blockfile.Close()

  scanner := reverse.NewScanner(blockfile)
  scanner.Split(bufio.ScanLines)

  var block Block
  retcode := scanner.Scan()

  if !retcode {
    return block, errors.New("No lines read from blockchain")
  }

  line := scanner.Text()

  err := json.Unmarshal([]byte(line), &block)
  if err != nil {
    return block, err
  }

  return block, nil
}
