// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

const NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF = 2 // ResultsHeader, BlockProof

func EncodeHeaderWithProof(headerProof *gossipmessages.ResultsBlockHeaderWithProof) ([][]byte, error) {
	if headerProof == nil || headerProof.Header == nil || headerProof.BlockProof == nil {
		return nil, errors.Errorf("codec failed to encode header with proof due to missing fields: %s", headerProof.String())
	}

	payloads := make([][]byte, 0, NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF)
	payloads = append(payloads, headerProof.Header.Raw())
	payloads = append(payloads, headerProof.BlockProof.Raw())

	return payloads, nil
}

func EncodeHeadersWithProofs(headersProofs []*gossipmessages.ResultsBlockHeaderWithProof) ([][]byte, error) {
	var payloads [][]byte

	for _, headerProof := range headersProofs {
		headerProofPayloads, err := EncodeHeaderWithProof(headerProof)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, headerProofPayloads...)
	}

	return payloads, nil
}

func DecodeHeaderWithProof(payloads [][]byte) (*gossipmessages.ResultsBlockHeaderWithProof, error) {
	results, err := DecodeHeadersWithProofs(payloads)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("codec failed to decode at least one header with proof")
	}

	return results[0], nil
}

func DecodeHeadersWithProofs(payloads [][]byte) (results []*gossipmessages.ResultsBlockHeaderWithProof, err error) {
	payloadIndex := uint32(0)

	for payloadIndex < uint32(len(payloads)) {
		if uint32(len(payloads)) < payloadIndex+NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF {
			return nil, errors.Errorf("codec failed to decode header with proof, missing payloads %d", len(payloads))
		}
		rxBlockHeader := protocol.ResultsBlockHeaderReader(payloads[payloadIndex])
		rxBlockProof := protocol.ResultsBlockProofReader(payloads[payloadIndex+1])
		payloadIndex += uint32(NUM_HARDCODED_PAYLOADS_FOR_HEADER_WITH_PROOF)

		headerProof := &gossipmessages.ResultsBlockHeaderWithProof{
			Header: 	rxBlockHeader,
			BlockProof: rxBlockProof,
		}

		results = append(results, headerProof)
	}
	return results, nil
}
