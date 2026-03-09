package primitives

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Eth1Data represents an Ethereum 1.x deposit data container.
// This is a native Go struct replacement for the proto-generated Eth1Data.
type Eth1Data struct {
	DepositRoot  [32]byte `ssz-size:"32"`
	DepositCount uint64
	BlockHash    [32]byte `ssz-size:"32"`
}

// MarshalJSON implements Beacon API JSON encoding.
// []byte → "0x" hex string, uint64 → decimal string.
func (e *Eth1Data) MarshalJSON() ([]byte, error) {
	type jsonEth1Data struct {
		DepositRoot  string `json:"deposit_root"`
		DepositCount string `json:"deposit_count"`
		BlockHash    string `json:"block_hash"`
	}
	return json.Marshal(&jsonEth1Data{
		DepositRoot:  hexutil.Encode(e.DepositRoot[:]),
		DepositCount: fmt.Sprintf("%d", e.DepositCount),
		BlockHash:    hexutil.Encode(e.BlockHash[:]),
	})
}

// UnmarshalJSON implements Beacon API JSON decoding.
func (e *Eth1Data) UnmarshalJSON(data []byte) error {
	type jsonEth1Data struct {
		DepositRoot  string `json:"deposit_root"`
		DepositCount string `json:"deposit_count"`
		BlockHash    string `json:"block_hash"`
	}
	var dec jsonEth1Data
	if err := json.Unmarshal(data, &dec); err != nil {
		return err
	}
	root, err := hexutil.Decode(dec.DepositRoot)
	if err != nil {
		return fmt.Errorf("invalid deposit_root: %w", err)
	}
	copy(e.DepositRoot[:], root)

	count, err := strconv.ParseUint(dec.DepositCount, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid deposit_count: %w", err)
	}
	e.DepositCount = count

	hash, err := hexutil.Decode(dec.BlockHash)
	if err != nil {
		return fmt.Errorf("invalid block_hash: %w", err)
	}
	copy(e.BlockHash[:], hash)
	return nil
}

// Copy returns a deep copy of Eth1Data.
func (e *Eth1Data) Copy() *Eth1Data {
	if e == nil {
		return nil
	}
	return &Eth1Data{
		DepositRoot:  e.DepositRoot,
		DepositCount: e.DepositCount,
		BlockHash:    e.BlockHash,
	}
}
