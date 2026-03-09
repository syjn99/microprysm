package eth_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

func TestEth1Data_MarshalJSON(t *testing.T) {
	depositRoot := make([]byte, 32)
	blockHash := make([]byte, 32)
	for i := range depositRoot {
		depositRoot[i] = byte(i)
	}
	for i := range blockHash {
		blockHash[i] = byte(0xff - i)
	}

	e := &eth.Eth1Data{
		DepositRoot:  depositRoot,
		DepositCount: 42,
		BlockHash:    blockHash,
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// Unmarshal into a generic map to verify field values.
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	// Check field names match Beacon API spec (snake_case).
	for _, key := range []string{"deposit_root", "deposit_count", "block_hash"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}

	// Check hex encoding.
	if got, want := m["deposit_root"].(string), hexutil.Encode(depositRoot); got != want {
		t.Errorf("deposit_root = %q, want %q", got, want)
	}
	if got, want := m["block_hash"].(string), hexutil.Encode(blockHash); got != want {
		t.Errorf("block_hash = %q, want %q", got, want)
	}

	// Check uint64 as decimal string.
	if got, want := m["deposit_count"].(string), "42"; got != want {
		t.Errorf("deposit_count = %q, want %q", got, want)
	}
}

func TestEth1Data_UnmarshalJSON(t *testing.T) {
	input := `{
		"deposit_root": "0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		"deposit_count": "42",
		"block_hash": "0xfffefdfcfbfaf9f8f7f6f5f4f3f2f1f0efeeedecebeae9e8e7e6e5e4e3e2e1e0"
	}`

	var e eth.Eth1Data
	if err := json.Unmarshal([]byte(input), &e); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}

	// Verify bytes.
	wantRoot := make([]byte, 32)
	for i := range wantRoot {
		wantRoot[i] = byte(i)
	}
	if got := e.DepositRoot; !bytesEqual(got, wantRoot) {
		t.Errorf("DepositRoot = %x, want %x", got, wantRoot)
	}

	wantHash := make([]byte, 32)
	for i := range wantHash {
		wantHash[i] = byte(0xff - i)
	}
	if got := e.BlockHash; !bytesEqual(got, wantHash) {
		t.Errorf("BlockHash = %x, want %x", got, wantHash)
	}

	// Verify uint64.
	if e.DepositCount != 42 {
		t.Errorf("DepositCount = %d, want 42", e.DepositCount)
	}
}

func TestEth1Data_RoundTrip(t *testing.T) {
	original := &eth.Eth1Data{
		DepositRoot:  bytes32(0xaa),
		DepositCount: 123456789,
		BlockHash:    bytes32(0xbb),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded eth.Eth1Data
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !bytesEqual(original.DepositRoot, decoded.DepositRoot) {
		t.Errorf("DepositRoot mismatch after round-trip")
	}
	if original.DepositCount != decoded.DepositCount {
		t.Errorf("DepositCount mismatch: %d vs %d", original.DepositCount, decoded.DepositCount)
	}
	if !bytesEqual(original.BlockHash, decoded.BlockHash) {
		t.Errorf("BlockHash mismatch after round-trip")
	}
}

// TestEth1Data_MatchesStructsOutput verifies that the proto MarshalJSON output
// is identical to what the existing api/server/structs conversion produces.
func TestEth1Data_MatchesStructsOutput(t *testing.T) {
	root := bytes32(0x11)
	hash := bytes32(0x22)

	original := &eth.Eth1Data{
		DepositRoot:  root,
		DepositCount: 999,
		BlockHash:    hash,
	}

	// Marshal using the new MarshalJSON.
	protoJSON, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("proto MarshalJSON: %v", err)
	}

	// Build the equivalent structs output manually (same logic as Eth1DataFromConsensus).
	type structsEth1Data struct {
		DepositRoot  string `json:"deposit_root"`
		DepositCount string `json:"deposit_count"`
		BlockHash    string `json:"block_hash"`
	}
	structs := structsEth1Data{
		DepositRoot:  hexutil.Encode(root),
		DepositCount: fmt.Sprintf("%d", original.DepositCount),
		BlockHash:    hexutil.Encode(hash),
	}
	structsJSON, err := json.Marshal(structs)
	if err != nil {
		t.Fatalf("structs Marshal: %v", err)
	}

	if string(protoJSON) != string(structsJSON) {
		t.Errorf("output mismatch:\nproto:   %s\nstructs: %s", protoJSON, structsJSON)
	}
}

func TestEth1Data_UnmarshalJSON_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"bad json", `{not json}`},
		{"bad deposit_root hex", `{"deposit_root":"not-hex","deposit_count":"1","block_hash":"0x00"}`},
		{"bad deposit_count", `{"deposit_root":"0x00","deposit_count":"not-a-number","block_hash":"0x00"}`},
		{"bad block_hash hex", `{"deposit_root":"0x00","deposit_count":"1","block_hash":"not-hex"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var e eth.Eth1Data
			if err := json.Unmarshal([]byte(tc.input), &e); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// bytes32 returns a 32-byte slice filled with b.
func bytes32(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
