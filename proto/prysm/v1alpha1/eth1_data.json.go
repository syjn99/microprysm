package eth

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// MarshalJSON implements Beacon API JSON encoding for the proto Eth1Data type.
func (e *Eth1Data) MarshalJSON() ([]byte, error) {
	type jsonEth1Data struct {
		DepositRoot  string `json:"deposit_root"`
		DepositCount string `json:"deposit_count"`
		BlockHash    string `json:"block_hash"`
	}
	return json.Marshal(&jsonEth1Data{
		DepositRoot:  hexutil.Encode(e.DepositRoot),
		DepositCount: fmt.Sprintf("%d", e.DepositCount),
		BlockHash:    hexutil.Encode(e.BlockHash),
	})
}

// UnmarshalJSON implements Beacon API JSON decoding for the proto Eth1Data type.
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
	e.DepositRoot = root

	count, err := strconv.ParseUint(dec.DepositCount, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid deposit_count: %w", err)
	}
	e.DepositCount = count

	hash, err := hexutil.Decode(dec.BlockHash)
	if err != nil {
		return fmt.Errorf("invalid block_hash: %w", err)
	}
	e.BlockHash = hash
	return nil
}
