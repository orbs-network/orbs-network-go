package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
)

type Config interface {
	NodeId() string
}

type levelDbBlockPersistence struct {
	blockWritten chan bool
	blockPairs   []*protocol.BlockPairContainer
	config       Config
	db *leveldb.DB
}

type config struct {
	name string
}

func (c *config) NodeId() string {
	return c.name
}

func NewLevelDbBlockPersistenceConfig(name string) Config {
	return &config{name: name}
}

func NewLevelDbBlockPersistence(config Config) BlockPersistence {
	db, err := leveldb.OpenFile("/tmp/db", nil)

	if err != nil{
		fmt.Println("Could not instantiate leveldb", err)
		panic("Could not instantiate leveldb")
	}

	return &levelDbBlockPersistence{
		config:       config,
		blockWritten: make(chan bool, 10),
		db: db,
	}
}

func (bp *levelDbBlockPersistence) WriteBlock(blockPair *protocol.BlockPairContainer) {
	key := "transaction-block-header-" + strconv.FormatUint(uint64(blockPair.TransactionsBlock.Header.BlockHeight()), 10)
	value := blockPair.TransactionsBlock.Header.Raw()

	fmt.Printf("Writing key %v, value %v\n", key, value)
	bp.db.Put([]byte(key), value, nil)

	bp.blockPairs = append(bp.blockPairs, blockPair)
	bp.blockWritten <- true
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	var results []*protocol.BlockPairContainer

	iter := bp.db.NewIterator(util.BytesPrefix([]byte("transaction-block-header-")), nil)

	for iter.Next()  {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		key := string(iter.Key())
		value := iter.Value()

		fmt.Printf("Retrieving key %v, value %v\n", key, value)

		transactionsBlock := &protocol.TransactionsBlockContainer{
			Header: protocol.TransactionsBlockHeaderReader(value),
		}

		fmt.Println("Height, timestamp", transactionsBlock.Header.BlockHeight(), transactionsBlock.Header.Timestamp())

		resultsBlock := &protocol.ResultsBlockContainer{}

		container := &protocol.BlockPairContainer{
			TransactionsBlock: transactionsBlock,
			ResultsBlock: resultsBlock,
		}

		results = append(results, container)
	}
	iter.Release()
	_ = iter.Error()

	return results
}
