// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package protocol

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
)

type EventJSON struct {
	ContractName string
	EventName    string
	Tuid         string
	EthAddress   string
	OrbsAddress  string
	Amount       string
}

type TransactionReceiptJSON struct {
	ExecutionResult string
}

type ResultsBlockHeaderJSON struct {
	ProtocolVersion  string
	VirtualChainId   string
	Timestamp        string
	ReceiptsRootHash string
}

type BlockRefJSON struct {
	MessageType string
	BlockHash   string
}

type SenderSignatureJSON struct {
	MemberId  string
	Signature string
}

type ResultsBlockProofJSON struct {
	TransactionsBlockHash string
	Signatures            []*SenderSignatureJSON
}

type TransactionReceiptProofJSON struct {
	Event                 *EventJSON
	RawEvent              string
	TransactionReceipt    *TransactionReceiptJSON
	RawTransactionReceipt string
	ResultsBlockHeader    *ResultsBlockHeaderJSON
	RawResultsBlockHeader string
	BlockRef              *BlockRefJSON
	RawBlockRef           string
	ResultsBlockProof     *ResultsBlockProofJSON
	RawResultsBlockProof  string
	ReceiptMerkleProof    []string
	RawPackedReceiptProof string
}

// this test currently just creates a json that we provide to Solidity unit tests that verify Solidity conforms to this contract
// TODO(v1): figure out what's the best way to automatically bring this json to the Solidity project for automated testing (now it's manual)
func TestTransactionReceiptProof(t *testing.T) {

	// event
	tuid := uint64(77393)
	ethAddress := common.FromHex("e128846cd5b4979d68a8c58a9bdfeee657b34de7")
	orbsAddress := common.FromHex("aa98846cd5b4979d68a8c58a9bdfeee657b34de7")
	amount := uint64(1202)
	eventBuilder := &protocol.EventBuilder{
		ContractName:        "asb_ether",
		EventName:           "OrbsTransferredOut",
		OutputArgumentArray: builders.PackedArgumentArrayEncode(tuid, orbsAddress, ethAddress, amount),
	}

	// transaction receipt
	transactionReceiptBuilder := &protocol.TransactionReceiptBuilder{
		Txhash:              hash.CalcSha256([]byte{0x12}),
		ExecutionResult:     protocol.EXECUTION_RESULT_SUCCESS,
		OutputArgumentArray: builders.PackedArgumentArrayEncode(),
		OutputEventsArray:   builders.PackedEventsArrayEncode([]*protocol.EventBuilder{eventBuilder}),
	}

	// results block header
	severalReceipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().Build(),
		builders.TransactionReceipt().Build(),
		transactionReceiptBuilder.Build(),
		builders.TransactionReceipt().Build(),
	}
	interestingReceiptIndex := 2
	receiptsRootHash := merkle.CalculateOrderedTreeRoot(digest.CalcReceiptHashes(severalReceipts))
	resultsBlockHeaderBuilder := &protocol.ResultsBlockHeaderBuilder{
		ProtocolVersion:                 1,
		VirtualChainId:                  42,
		BlockHeight:                     8234,
		PrevBlockHashPtr:                hash.CalcSha256([]byte{0x11}),
		Timestamp:                       15527734,
		ReceiptsMerkleRootHash:          receiptsRootHash,
		StateDiffHash:                   hash.CalcSha256([]byte{0x33}),
		TransactionsBlockHashPtr:        hash.CalcSha256([]byte{0x44}),
		PreExecutionStateMerkleRootHash: hash.CalcSha256([]byte{0x55}),
		NumTransactionReceipts:          12,
		NumContractStateDiffs:           34,
	}

	// results block proof
	blockHash := hash.CalcSha256(resultsBlockHeaderBuilder.TransactionsBlockHashPtr, hash.CalcSha256(resultsBlockHeaderBuilder.Build().Raw()))
	blockRefBuilder := &lhprotocol.BlockRefBuilder{
		MessageType: lhprotocol.LEAN_HELIX_COMMIT,
		BlockHeight: lhprimitives.BlockHeight(resultsBlockHeaderBuilder.BlockHeight),
		View:        2893478,
		BlockHash:   lhprimitives.BlockHash(blockHash),
	}
	dataToSign := blockRefBuilder.Build().Raw()
	nodeSignatures := []*lhprotocol.SenderSignatureBuilder{}
	signaturesJSON := []*SenderSignatureJSON{}
	for i := 0; i < 5; i++ {
		kp := testKeys.EcdsaSecp256K1KeyPairForTests(i)
		sig, err := digest.SignAsNode(kp.PrivateKey(), dataToSign)
		require.NoError(t, err)
		nodeSignatures = append(nodeSignatures, &lhprotocol.SenderSignatureBuilder{
			MemberId:  lhprimitives.MemberId(kp.NodeAddress()),
			Signature: lhprimitives.Signature(sig),
		})
		signaturesJSON = append(signaturesJSON, &SenderSignatureJSON{
			MemberId:  bytesToJSON(kp.NodeAddress()),
			Signature: bytesToJSON(sig),
		})
	}
	leanHelixBlockProofBuilder := &lhprotocol.BlockProofBuilder{
		BlockRef:            blockRefBuilder,
		Nodes:               nodeSignatures,
		RandomSeedSignature: lhprimitives.RandomSeedSignature(hash.CalcSha256([]byte{0x39})),
	}
	resultsBlockProofBuilder := &protocol.ResultsBlockProofBuilder{
		TransactionsBlockHash: resultsBlockHeaderBuilder.TransactionsBlockHashPtr,
		Type:                  protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix:             leanHelixBlockProofBuilder.Build().Raw(),
	}

	// receipt merkle proof
	receiptMerkleProof, err := merkle.NewOrderedTree(digest.CalcReceiptHashes(severalReceipts)).GetProof(interestingReceiptIndex)
	require.NoError(t, err)

	// receipt proof
	receiptProofBuilder := &protocol.ReceiptProofBuilder{
		Header:       resultsBlockHeaderBuilder,
		BlockProof:   resultsBlockProofBuilder,
		ReceiptProof: merkle.FlattenOrderedTreeProof(receiptMerkleProof),
	}

	// wrap everything up
	transactionReceiptProofJson := &TransactionReceiptProofJSON{
		Event: &EventJSON{
			ContractName: string(eventBuilder.ContractName),
			EventName:    string(eventBuilder.EventName),
			Tuid:         numberToJSON(tuid),
			EthAddress:   bytesToJSON(ethAddress),
			OrbsAddress:  bytesToJSON(orbsAddress),
			Amount:       numberToJSON(amount),
		},
		RawEvent: bytesToJSON(eventBuilder.Build().Raw()),
		TransactionReceipt: &TransactionReceiptJSON{
			ExecutionResult: numberToJSON(transactionReceiptBuilder.ExecutionResult),
		},
		RawTransactionReceipt: bytesToJSON(transactionReceiptBuilder.Build().Raw()),
		ResultsBlockHeader: &ResultsBlockHeaderJSON{
			ProtocolVersion:  numberToJSON(resultsBlockHeaderBuilder.ProtocolVersion),
			VirtualChainId:   numberToJSON(resultsBlockHeaderBuilder.VirtualChainId),
			Timestamp:        numberToJSON(resultsBlockHeaderBuilder.Timestamp),
			ReceiptsRootHash: bytesToJSON(resultsBlockHeaderBuilder.ReceiptsMerkleRootHash),
		},
		RawResultsBlockHeader: bytesToJSON(resultsBlockHeaderBuilder.Build().Raw()),
		BlockRef: &BlockRefJSON{
			MessageType: numberToJSON(blockRefBuilder.MessageType),
			BlockHash:   bytesToJSON(blockRefBuilder.BlockHash),
		},
		RawBlockRef: bytesToJSON(blockRefBuilder.Build().Raw()),
		ResultsBlockProof: &ResultsBlockProofJSON{
			TransactionsBlockHash: bytesToJSON(resultsBlockHeaderBuilder.TransactionsBlockHashPtr),
			Signatures:            signaturesJSON,
		},
		RawResultsBlockProof:  bytesToJSON(resultsBlockProofBuilder.Build().Raw()),
		ReceiptMerkleProof:    merkleProofToJSON(receiptMerkleProof),
		RawPackedReceiptProof: bytesToJSON(receiptProofBuilder.Build().Raw()),
	}
	jsonBytes, err := json.MarshalIndent(transactionReceiptProofJson, "", "  ")
	require.NoError(t, err)
	jsonString := string(jsonBytes)
	t.Log(jsonString)

	// write json
	ioutil.WriteFile("TransactionReceiptProof.json", jsonBytes, 0655)

	// print some hex dumps
	fmt.Printf("\nEvent:\n")
	eventBuilder.HexDump("", 0)

	fmt.Printf("\nTransactionReceipt:\n")
	transactionReceiptBuilder.HexDump("", 0)

	fmt.Printf("\nResultsBlockHeader:\n")
	resultsBlockHeaderBuilder.HexDump("", 0)

	fmt.Printf("\nLeanHelixBlockProof:\n")
	leanHelixBlockProofBuilder.HexDump("", 0)

	fmt.Printf("\nResultsBlockProof:\n")
	resultsBlockProofBuilder.HexDump("", 0)

	fmt.Printf("\nReceiptProof:\n")
	receiptProofBuilder.HexDump("", 0)

	fmt.Println()
}
