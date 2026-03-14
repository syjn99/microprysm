# Proto Field Alignment — Implementation Plan

## Strategy

Use `ethereum.eth.ext.spec_name` proto annotations to declare spec-compliant JSON field names. This avoids renaming proto fields and the resulting thousands of Go code changes.

## Step 1: Add spec_name annotations (this PR) ✅

Add `spec_name` to all 22 discrepant proto fields. No Go code changes needed. Proto field names and Go struct field names remain unchanged.

## Step 2: Update prysm-jsongen to read spec_name

`prysm-jsongen` currently reads JSON field names from `.pb.go` struct tags (`json:"field_name"`). It needs to be updated to:

1. Parse proto files to extract `spec_name` annotations
2. When `spec_name` is present, use it as the JSON key instead of the proto-derived `json_name`
3. Fall back to `json_name` when no `spec_name` is set

**Implementation options:**
- **Option A:** Parse `.proto` files directly in jsongen (adds proto file dependency)
- **Option B:** Parse the generated `.pb.go` for `protobuf:"..."` struct tags which contain field options
- **Option C:** Use `protoreflect` at runtime to read field options from the compiled proto descriptor

## Step 3: Verify oracle tests pass

Re-run `prysm-jsongen` oracle comparison tests to confirm all 22 discrepancies are resolved.

## Future: Full proto field rename (optional)

Once microprysm is validated and stable, proto fields can be renamed to match spec names directly. This would make the codebase more intuitive but requires ~3,900 Go reference updates. The `spec_name` approach makes this optional rather than blocking.
