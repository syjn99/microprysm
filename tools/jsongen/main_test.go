package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStruct_Eth1Data(t *testing.T) {
	// Use the actual proto file in the repo.
	dir := filepath.Join("..", "..", "proto", "prysm", "v1alpha1")
	fields, pkgName, err := parseStruct(dir, "Eth1Data")
	if err != nil {
		t.Fatalf("parseStruct: %v", err)
	}
	if pkgName != "eth" {
		t.Errorf("package = %q, want %q", pkgName, "eth")
	}
	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}

	want := []fieldInfo{
		{Name: "DepositRoot", JSONTag: "deposit_root", Kind: "bytes"},
		{Name: "DepositCount", JSONTag: "deposit_count", Kind: "uint64"},
		{Name: "BlockHash", JSONTag: "block_hash", Kind: "bytes"},
	}
	for i, f := range fields {
		if f != want[i] {
			t.Errorf("field[%d] = %+v, want %+v", i, f, want[i])
		}
	}
}

func TestGenerate_Eth1Data(t *testing.T) {
	fields := []fieldInfo{
		{Name: "DepositRoot", JSONTag: "deposit_root", Kind: "bytes"},
		{Name: "DepositCount", JSONTag: "deposit_count", Kind: "uint64"},
		{Name: "BlockHash", JSONTag: "block_hash", Kind: "bytes"},
	}
	code, err := generate("eth", "Eth1Data", fields)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	src := string(code)

	// Verify key fragments are present.
	checks := []string{
		"func (x *Eth1Data) MarshalJSON()",
		"func (x *Eth1Data) UnmarshalJSON(enc []byte) error",
		`hexutil.Encode(x.DepositRoot)`,
		`fmt.Sprintf("%d", x.DepositCount)`,
		`hexutil.Decode(dec.DepositRoot)`,
		`strconv.ParseUint(dec.DepositCount, 10, 64)`,
		`json:"deposit_root"`,
		`json:"block_hash"`,
	}
	for _, c := range checks {
		if !contains(src, c) {
			t.Errorf("generated code missing %q", c)
		}
	}
}

func TestGenerate_WritesFile(t *testing.T) {
	dir := t.TempDir()
	// Create a minimal .pb.go source to parse.
	src := `package foo

import "google.golang.org/protobuf/runtime/protoimpl"

type MyMsg struct {
	state         protoimpl.MessageState
	Value         []byte ` + "`" + `protobuf:"bytes,1,opt" json:"value,omitempty"` + "`" + `
	Count         uint64 ` + "`" + `protobuf:"varint,2,opt" json:"count,omitempty"` + "`" + `
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}
`
	if err := os.WriteFile(filepath.Join(dir, "test.pb.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	fields, pkgName, err := parseStruct(dir, "MyMsg")
	if err != nil {
		t.Fatalf("parseStruct: %v", err)
	}
	if pkgName != "foo" {
		t.Errorf("pkg = %q, want foo", pkgName)
	}
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}

	code, err := generate(pkgName, "MyMsg", fields)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "my_msg.json.go")
	if err := os.WriteFile(out, code, 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Eth1Data", "eth1_data"},
		{"DepositRoot", "deposit_root"},
		{"BlockHash", "block_hash"},
		{"MyMsg", "my_msg"},
	}
	for _, tc := range tests {
		got := camelToSnake(tc.in)
		if got != tc.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
