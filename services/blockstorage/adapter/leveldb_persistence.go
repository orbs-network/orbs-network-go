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

	err := bp.db.Put([]byte(key), value, nil)

	if err == nil {
		bp.blockWritten <- true
	} else {
		fmt.Println("Failed to write block", err)
	}
}

func constructBlockFromStorage(data []byte) *protocol.BlockPairContainer {
	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: protocol.TransactionsBlockHeaderReader(data),
	}

	resultsBlock := &protocol.ResultsBlockContainer{}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock: resultsBlock,
	}

	return container
}

func (bp *levelDbBlockPersistence) ReadAllBlocks() []*protocol.BlockPairContainer {
	var results []*protocol.BlockPairContainer

	iter := bp.db.NewIterator(util.BytesPrefix([]byte("transaction-block-header-")), nil)

	for iter.Next()  {
		key := string(iter.Key())
		data := make([]byte, len(iter.Value()))
		copy(data, iter.Value())

		fmt.Printf("Retrieving key %v, value %v\n", key, data)

		results = append(results, constructBlockFromStorage(data))
	}
	iter.Release()
	_ = iter.Error()

	return results
}
