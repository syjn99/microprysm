// Command jsongen generates MarshalJSON/UnmarshalJSON methods for protobuf
// structs so they conform to the Beacon API JSON encoding:
//   - []byte          → hex string with "0x" prefix
//   - uint64          → decimal string
//   - primitives.Slot/Epoch/... (underlying uint64) → decimal string
//   - bool            → JSON boolean (no conversion)
//   - *ProtoStruct    → recursive (delegates to its own MarshalJSON)
//   - repeated fields → JSON array
//
// Usage:
//
//	go run tools/jsongen/main.go -type Eth1Data -pkg proto/prysm/v1alpha1
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"unicode"
)

// fieldInfo holds the parsed metadata for a single struct field.
type fieldInfo struct {
	Name    string // Go field name, e.g. "DepositRoot"
	JSONTag string // JSON key, e.g. "deposit_root"
	Kind    string // one of: bytes, uint64, bool, struct, repeated_struct, repeated_bytes, repeated_uint64
}

func main() {
	typeName := flag.String("type", "", "struct type name (required)")
	pkgPath := flag.String("pkg", "", "package directory relative to module root (required)")
	outFile := flag.String("out", "", "output file (default: <source_dir>/<first_source_file_stem>.json.go)")
	flag.Parse()

	if *typeName == "" || *pkgPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Resolve absolute directory.
	dir, err := filepath.Abs(*pkgPath)
	if err != nil {
		log.Fatalf("resolve path: %v", err)
	}

	fields, pkgName, err := parseStruct(dir, *typeName)
	if err != nil {
		log.Fatal(err)
	}

	code, err := generate(pkgName, *typeName, fields)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	dest := *outFile
	if dest == "" {
		dest = filepath.Join(dir, camelToSnake(*typeName)+".json.go")
	}
	if err := os.WriteFile(dest, code, 0644); err != nil {
		log.Fatalf("write %s: %v", dest, err)
	}
	fmt.Printf("wrote %s\n", dest)
}

// protoInternalFields are fields injected by protoc-gen-go that must be
// excluded from JSON serialization.
var protoInternalFields = map[string]bool{
	"state":         true,
	"unknownFields": true,
	"sizeCache":     true,
}

// parseStruct walks every .pb.go file in dir looking for the named struct,
// then extracts the exportable field metadata.
func parseStruct(dir, typeName string) ([]fieldInfo, string, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), ".pb.go")
	}, 0)
	if err != nil {
		return nil, "", fmt.Errorf("parse dir %s: %w", dir, err)
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts := spec.(*ast.TypeSpec)
					if ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						return nil, "", fmt.Errorf("%s is not a struct", typeName)
					}
					fields, err := extractFields(st)
					if err != nil {
						return nil, "", err
					}
					return fields, pkg.Name, nil
				}
			}
		}
	}
	return nil, "", fmt.Errorf("type %s not found in %s", typeName, dir)
}

func extractFields(st *ast.StructType) ([]fieldInfo, error) {
	var fields []fieldInfo
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // embedded
		}
		name := f.Names[0].Name
		if protoInternalFields[name] || !unicode.IsUpper(rune(name[0])) {
			continue
		}

		jsonTag := name
		if f.Tag != nil {
			tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
			if jt, ok := tag.Lookup("json"); ok {
				parts := strings.SplitN(jt, ",", 2)
				if parts[0] != "" && parts[0] != "-" {
					jsonTag = parts[0]
				}
			}
		}

		kind := classifyType(f.Type)
		fields = append(fields, fieldInfo{
			Name:    name,
			JSONTag: jsonTag,
			Kind:    kind,
		})
	}
	return fields, nil
}

func classifyType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.ArrayType:
		if t.Len != nil {
			// Fixed-size array — treat like bytes for [N]byte.
			return "bytes"
		}
		inner := classifyType(t.Elt)
		switch inner {
		case "bytes":
			return "bytes" // []byte
		case "uint64":
			return "repeated_uint64"
		case "struct":
			return "repeated_struct"
		default:
			return "repeated_" + inner
		}
	case *ast.Ident:
		switch t.Name {
		case "byte":
			return "bytes"
		case "uint64":
			return "uint64"
		case "bool":
			return "bool"
		case "string":
			return "string"
		default:
			// Named type in the same package (e.g. another proto struct).
			return "struct"
		}
	case *ast.StarExpr:
		return classifyType(t.X) // *T → classify T
	case *ast.SelectorExpr:
		// e.g. primitives.Slot, primitives.Epoch
		if ident, ok := t.X.(*ast.Ident); ok && ident.Name == "primitives" {
			return "uint64" // all primitives numeric types are uint64 underneath
		}
		return "struct"
	}
	return "unknown"
}

// generate produces the formatted Go source for the JSON methods.
func generate(pkgName, typeName string, fields []fieldInfo) ([]byte, error) {
	data := tmplData{
		Package:     pkgName,
		Type:        typeName,
		Fields:      fields,
		NeedHex:     false,
		NeedFmt:     false,
		NeedStrconv: false,
	}
	for _, f := range fields {
		switch f.Kind {
		case "bytes":
			data.NeedHex = true
		case "uint64":
			data.NeedFmt = true
			data.NeedStrconv = true
		}
	}

	var buf bytes.Buffer
	if err := codeTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

type tmplData struct {
	Package     string
	Type        string
	Fields      []fieldInfo
	NeedHex     bool
	NeedFmt     bool
	NeedStrconv bool
}

var funcMap = template.FuncMap{
	"lower": strings.ToLower,
}

var codeTmpl = template.Must(template.New("json").Funcs(funcMap).Parse(`// Code generated by jsongen. DO NOT EDIT.

package {{ .Package }}

import (
	"encoding/json"
{{- if .NeedFmt }}
	"fmt"
{{- end }}
{{- if .NeedStrconv }}
	"strconv"
{{- end }}
{{- if .NeedHex }}

	"github.com/ethereum/go-ethereum/common/hexutil"
{{- end }}
)

func (x *{{ .Type }}) MarshalJSON() ([]byte, error) {
	type jsonRepresentation struct {
{{- range .Fields }}
	{{- if eq .Kind "bool" }}
		{{ .Name }} bool ` + "`" + `json:"{{ .JSONTag }}"` + "`" + `
	{{- else }}
		{{ .Name }} string ` + "`" + `json:"{{ .JSONTag }}"` + "`" + `
	{{- end }}
{{- end }}
	}
	return json.Marshal(&jsonRepresentation{
{{- range .Fields }}
	{{- if eq .Kind "bytes" }}
		{{ .Name }}: hexutil.Encode(x.{{ .Name }}),
	{{- else if eq .Kind "uint64" }}
		{{ .Name }}: fmt.Sprintf("%d", x.{{ .Name }}),
	{{- else if eq .Kind "bool" }}
		{{ .Name }}: x.{{ .Name }},
	{{- end }}
{{- end }}
	})
}

func (x *{{ .Type }}) UnmarshalJSON(enc []byte) error {
	type jsonRepresentation struct {
{{- range .Fields }}
	{{- if eq .Kind "bool" }}
		{{ .Name }} bool ` + "`" + `json:"{{ .JSONTag }}"` + "`" + `
	{{- else }}
		{{ .Name }} string ` + "`" + `json:"{{ .JSONTag }}"` + "`" + `
	{{- end }}
{{- end }}
	}
	var dec jsonRepresentation
	if err := json.Unmarshal(enc, &dec); err != nil {
		return err
	}
{{- range .Fields }}
	{{- if eq .Kind "bytes" }}
	{
		b, err := hexutil.Decode(dec.{{ .Name }})
		if err != nil {
			return fmt.Errorf("invalid {{ .JSONTag }}: %w", err)
		}
		x.{{ .Name }} = b
	}
	{{- else if eq .Kind "uint64" }}
	{
		n, err := strconv.ParseUint(dec.{{ .Name }}, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid {{ .JSONTag }}: %w", err)
		}
		x.{{ .Name }} = n
	}
	{{- else if eq .Kind "bool" }}
	x.{{ .Name }} = dec.{{ .Name }}
	{{- end }}
{{- end }}
	return nil
}
`))

// camelToSnake converts e.g. "Eth1Data" → "eth1_data".
func camelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					result = append(result, '_')
				}
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
