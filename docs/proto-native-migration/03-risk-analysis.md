# Risk Analysis & Verification Strategy

## Key Risks

### 1. SSZ Encoding Mismatch
**Risk:** Native struct SSZ encoding differs from proto SSZ encoding → consensus failure.
**Severity:** Critical — would cause chain split.
**Mitigation:**
- Round-trip test: `MarshalSSZ(native)` must equal `MarshalSSZ(proto)` for identical data
- HashTreeRoot must produce identical values
- Use spec test vectors from `consensus-spec-tests` to validate
- **Gate:** No type migration merges without SSZ round-trip proof

### 2. JSON Encoding Regression
**Risk:** Beacon API JSON format changes → breaks API consumers.
**Severity:** High — would break validators, explorers, tooling.
**Mitigation:**
- Diff test: `json.Marshal(native)` vs current `structs.XxxFromConsensus()` output
- Verify hex encoding for all `[]byte` fields
- Verify string encoding for all `uint64` fields (Beacon API uses quoted integers)
- Run API conformance tests

### 3. Mass Import Refactoring (1,110+ files)
**Risk:** Mechanical `ethpb.X` → `newpkg.X` changes introduce subtle bugs.
**Severity:** Medium — compiler catches most, but pointer/value semantics may differ.
**Mitigation:**
- Proto types use pointer receivers; native structs may use value receivers
- Use `gopls` rename or `sed` with manual review
- Batch by package to limit blast radius
- Each PR must pass full test suite

### 4. Mixed-Type Period
**Risk:** During migration, some fields are native while parent types are still proto.
**Severity:** Medium — conversion overhead and confusion.
**Mitigation:**
- Bottom-up wave order minimizes mixed period
- Temporary conversion helpers (native ↔ proto) during transition
- Clear naming: `primitives.Checkpoint` vs `ethpb.Checkpoint`
- Document which types are migrated in a tracking table

### 5. BeaconState Interface Breakage
**Risk:** `state.BeaconState` interface and `state/stateutil/` hash functions depend on proto field layout.
**Severity:** High — state management is critical.
**Mitigation:**
- Migrate BeaconState last (Wave 5)
- All leaf/composite types proven stable before touching state
- May need state interface refactor as prerequisite

### 6. DB Serialization
**Risk:** BoltDB stores SSZ-encoded proto types. Changing types may break DB reads.
**Severity:** High — would corrupt or fail to read existing databases.
**Mitigation:**
- SSZ encoding is schema-defined (field order + sizes), not type-name-dependent
- If native structs have identical SSZ schema, DB reads work transparently
- **Must verify:** field order in native struct matches proto field number order
- Add migration test: read existing DB → deserialize with native types

### 7. P2P Encoding
**Risk:** P2P layer uses SSZ for block/attestation gossip. Type changes could break interop.
**Severity:** Critical — would fork from network.
**Mitigation:**
- Same as Risk 1 — SSZ encoding must be bit-identical
- Interop test: gossip encoded by native type must be decodable by standard proto type

## Verification Strategy

### Per-PR Checklist
1. [ ] SSZ round-trip test passes (for types with SSZ)
2. [ ] JSON output matches existing format (for types with JSON)
3. [ ] `bazel build //...` passes
4. [ ] `bazel test //...` passes (or relevant packages)
5. [ ] BUILD.bazel updated via `gazelle`
6. [ ] Precheck scripts pass (gofmt, etc.)

### Phase Gates

| Phase | Gate Criteria |
|-------|--------------|
| Phase 1 | All REST responses unchanged; `conversions*.go` deleted; E2E pass |
| Phase 2 Wave 1 | Leaf types migrated; SSZ round-trip proven; unit tests pass |
| Phase 2 Wave 2-3 | Composite types migrated; attestation/block flows tested |
| Phase 2 Wave 4 | All block types migrated; block processing E2E pass |
| Phase 2 Wave 5 | BeaconState migrated; full E2E pass; DB migration tested |
| Phase 3 | Proto deleted; no `.pb.go` files; full E2E pass |

### Automated Tests to Add
- `TestSSZRoundTrip_NativeVsProto` — for each migrated type
- `TestJSONCompat_NativeVsStructs` — for each type with API JSON
- `TestDBRead_NativeTypes` — read existing DB fixtures with native types
- `TestP2PGossip_NativeTypes` — encode/decode gossip messages

## Estimated Timeline

| Phase | Duration (est.) | Parallelism |
|-------|----------------|-------------|
| Phase 1 | 1-2 weeks | Sequential (7 PRs) |
| Phase 2 Wave 1-3 | 2-3 weeks | Partially parallel within waves |
| Phase 2 Wave 4-5 | 3-4 weeks | Sequential (complex types) |
| Phase 3 | 1 week | Sequential (4 PRs) |
| **Total** | **7-10 weeks** | |

## Rollback Plan

Each phase is independently valuable:
- **Phase 1 only:** Still get ~8,500 lines removed. Proto types gain JSON methods.
- **Phase 2 partial:** Migrated types work; unmigrated types stay proto. No regression.
- **Phase 2 complete, Phase 3 skipped:** Native types used everywhere, proto files just unused baggage.

There is no point of no return until Phase 3 (proto deletion). Everything before that is additive/replacement.
