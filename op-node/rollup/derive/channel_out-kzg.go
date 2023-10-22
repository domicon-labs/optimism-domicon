package derive

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type proof struct {
	//Domicon Data
	Commitment [50]byte
	Signature  [64]byte
}

// BlockToBatch transforms a block into a batch object that can easily be RLP encoded.
func BlockToKZGBatch(block *types.Block) (*BatchData, L1BlockInfo, error) {
	opaqueTxs := make([]hexutil.Bytes, 0, len(block.Transactions()))

	for i, tx := range block.Transactions() {
		if tx.Type() == types.DepositTxType {
			continue
		}
		otx, err := tx.MarshalBinary()
		if err != nil {
			return nil, L1BlockInfo{}, fmt.Errorf("could not encode tx %v in block %v: %w", i, tx.Hash(), err)
		}
		opaqueTxs = append(opaqueTxs, otx)
	}

	if len(block.Transactions()) == 0 {
		return nil, L1BlockInfo{}, fmt.Errorf("block %v has no transactions", block.Hash())
	}
	l1InfoTx := block.Transactions()[0]
	if l1InfoTx.Type() != types.DepositTxType {
		return nil, L1BlockInfo{}, ErrNotDepositTx
	}
	l1Info, err := L1InfoDepositTxData(l1InfoTx.Data())
	if err != nil {
		return nil, l1Info, fmt.Errorf("could not parse the L1 Info deposit: %w", err)
	}

	batch := BatchV1{
		ParentHash:   block.ParentHash(),
		EpochNum:     rollup.Epoch(l1Info.Number),
		EpochHash:    l1Info.BlockHash,
		Timestamp:    block.Time(),
		Transactions: opaqueTxs,
	}

	commitment := GetCommitmentWithBatch(batch)
	signature := GetSignatureWithBatch(batch)

	proof := &proof{
		Commitment: commitment,
		Signature:  signature,
	}

	// 序列化为[]byte
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return nil, L1BlockInfo{}, fmt.Errorf("proof 序列化失败")
	}

	opaqueTxs = append(opaqueTxs, proofBytes)

	return NewSingularBatchData(
		SingularBatch{
			ParentHash:   batch.ParentHash,
			EpochNum:     batch.EpochNum,
			EpochHash:    batch.EpochHash,
			Timestamp:    batch.Timestamp,
			Transactions: opaqueTxs,
		},
	), l1Info, nil
}
