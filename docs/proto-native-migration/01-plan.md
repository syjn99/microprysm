# Proto → Native Go Struct Migration Plan

## Strategy: Three Phases

The migration proceeds in three sequential phases. Each phase is independently valuable and leaves the codebase in a working state.

```
Phase 1: jsongen on .pb.go         → Delete api/server/structs/ (~8,500 lines)
Phase 2: Native Go structs         → Replace ethpb.* with native types
Phase 3: Delete .proto + .pb.go    → Remove protobuf entirely (~103,000+ lines: proto definitions + generated Go)
```

---

## Phase 1: jsongen on .pb.go → Delete structs/ Conversions

**Goal:** Run `prysm-jsongen` on existing `.pb.go` files to generate `MarshalJSON`/`UnmarshalJSON` directly on proto types. Then delete all `api/server/structs/` conversion code.

**Why first:** This is the lowest-risk, highest-ROI step. No type changes, no import changes — just eliminating the duplicate JSON layer.

### PR 1.1: Add jsongen to proto build pipeline
**Scope:** Integrate `prysm-jsongen` into the build
- Add `prysm-jsongen` as a tool dependency
- Create `go generate` directives or Bazel rules to run jsongen on `.pb.go` files
- Generate `*_json.go` files alongside existing `.pb.go`
- Verify generated JSON matches existing `structs.XxxFromConsensus()` output
- **Risk:** Low — additive only, no existing code changed
- **Success criteria:** `prysm-jsongen` runs cleanly on all proto types, output matches spec

### PR 1.2: Migrate REST handlers — core types (Checkpoint, Fork, Eth1Data, etc.)
**Scope:** Wave 1 leaf types in REST handlers
- Replace `structs.CheckpointFromConsensus(proto)` → `json.Marshal(proto)` (now using generated methods)
- Replace `structs.Checkpoint{}; json.Unmarshal(body); checkpoint.ToConsensus()` → `json.Unmarshal(body, proto)`
- Update REST handler files in `beacon-chain/rpc/eth/`
- **Risk:** Low — leaf types with simple fields
- **Success criteria:** All REST API responses for these types remain spec-compliant

### PR 1.3: Migrate REST handlers — attestation types
**Scope:** AttestationData, Attestation, AttestationElectra, IndexedAttestation
- Same pattern as PR 1.2
- **Risk:** Low-Medium — attestations are high-traffic

### PR 1.4: Migrate REST handlers — block types
**Scope:** BeaconBlockHeader, SignedBeaconBlockHeader, BeaconBlock/Body (all forks)
- Largest batch — blocks have the most conversion code
- **Risk:** Medium — complex nested types, per-fork variants
- **Success criteria:** All `/eth/v2/beacon/blocks/*` responses unchanged

### PR 1.5: Migrate REST handlers — validator & state types
**Scope:** Validator, BeaconState query responses, duties responses
- **Risk:** Medium — validator duties are critical path

### PR 1.6: Migrate REST handlers — remaining types
**Scope:** LightClient, SyncCommittee, Blobs, EIP-7251, Gloas
- Catch-all for remaining conversions
- **Risk:** Low-Medium

### PR 1.7: Delete api/server/structs/ conversion code
**Scope:** Remove all `conversions*.go` files and unused struct types
- Only delete structs that are fully replaced by jsongen
- Keep structs still needed for request parsing (if any)
- Run `grep -r "structs\." --include="*.go"` to verify zero remaining references
- **Risk:** Low — all callers already migrated
- **Success criteria:** `api/server/structs/conversions*.go` deleted, ~8,500 lines removed

### Phase 1 Gate
- [ ] All REST API endpoints return spec-compliant JSON
- [ ] All unit tests pass
- [ ] E2E tests pass
- [ ] `api/server/structs/conversions*.go` deleted
- [ ] No `FromConsensus`/`ToConsensus` calls remain
- [ ] **Estimated removal: ~8,500 lines**

---

## Phase 2: Native Go Structs + sszgen + jsongen

**Goal:** Define native Go structs to replace proto message types. Generate SSZ and JSON encoding. Update all 1,110+ importers.

**Why second:** This is the heavy lift. Requires careful wave-by-wave migration following the dependency order.

### Wave 1 — Pure Leaf Types

#### PR 2.1: Native Checkpoint + Fork + VoluntaryExit
**Scope:** Define structs in `consensus-types/` (sub-packages by domain)
```go
type Checkpoint struct {
    Epoch primitives.Epoch     `ssz-size:"8"`
    Root  [32]byte             `ssz-size:"32"`
}
```
- Run `sszgen` → `*_encoding.go`
- Run `prysm-jsongen` → `*_json.go` (hex encoding for Root)
- Update all `ethpb.Checkpoint` → `primitives.Checkpoint` references
- Update BUILD.bazel deps
- **Risk:** Medium — first type migration sets the pattern. 9+9+1 = 19 parent references.
- **Success criteria:** SSZ encoding bit-identical to proto; all tests pass

#### PR 2.2: Native Eth1Data + BeaconBlockHeader
**Scope:** Two high-fan-out leaf types (20 + 11 parents)
- Same pattern as PR 2.1
- **Risk:** Medium — high parent count means many files touched

#### PR 2.3: Native SyncAggregate + SyncCommittee + Deposit
**Scope:** Remaining high-impact leaves (21 + 15 + 13 parents)
- **Risk:** Medium

#### PR 2.4: Native Validator + remaining Wave 1 leaves
**Scope:** Validator (8 fields, 9 parents), HistoricalSummary, BLSToExecutionChange
- **Risk:** Medium — Validator is complex (8 fields with various types)

### Wave 2 — 1-dep Types

#### PR 2.5: Native SignedVoluntaryExit + SignedBLSToExecutionChange
**Scope:** Signed wrapper types (11 + 7 parents)
- Depend on Wave 1 types (VoluntaryExit, BLSToExecutionChange)
- **Risk:** Low — simple wrapper structs

#### PR 2.6: Native AttestationData + SignedBeaconBlockHeader
**Scope:** Key composite types (6 + 3 parents)
- Depend on Checkpoint, BeaconBlockHeader
- **Risk:** Medium — AttestationData is heavily used

#### PR 2.7: Native LightClientHeader variants
**Scope:** LightClientHeaderDeneb and related (7 parents)
- Depend on BeaconBlockHeader
- **Risk:** Low — isolated subsystem

### Wave 3 — 2-dep Types

#### PR 2.8: Native ProposerSlashing + Attestation + AttestationElectra
**Scope:** Key consensus types (11 + 9 + 4 parents)
- Depend on SignedBeaconBlockHeader, AttestationData
- **Risk:** Medium — core consensus operations

#### PR 2.9: Native IndexedAttestation + remaining Wave 3
**Scope:** Remaining composite types
- **Risk:** Low-Medium

### Wave 4 — BeaconBlock Hierarchy

#### PR 2.10–2.16: Native BeaconBlockBody (per-fork)
**Scope:** One PR per fork variant (Phase0, Altair, Bellatrix, Capella, Deneb, Electra, Gloas)
- Each BlockBody depends on many Wave 1-3 types
- **Risk:** High — largest and most complex types, per-fork SSZ differences
- **Success criteria:** SSZ round-trip identical; block processing unchanged

#### PR 2.17–2.23: Native BeaconBlock + SignedBeaconBlock (per-fork)
**Scope:** Wrapper types around BlockBody
- **Risk:** Medium — straightforward wrappers once BlockBody is done

### Wave 5 — BeaconState

#### PR 2.24–2.30: Native BeaconState (per-fork)
**Scope:** One PR per fork variant
- Depends on virtually all other types
- **Risk:** High — state is the most critical data structure
- **Must be last** in Wave ordering

### Phase 2 Gate
- [ ] All proto message types replaced with native Go structs
- [ ] SSZ encoding bit-identical (verified by round-trip tests)
- [ ] JSON encoding spec-compliant (verified by spectests if available)
- [ ] All unit tests pass
- [ ] E2E tests pass
- [ ] No `ethpb.*` references remain for migrated types
- [ ] **Estimated: ~30 PRs, touching 1,110+ files**

---

## Phase 3: Delete .proto + .pb.go

**Goal:** Remove all protobuf definitions and generated code. Remove `google.golang.org/protobuf` dependency.

### PR 3.1: Delete proto service definitions (if any remain)
**Scope:** Remove any remaining `service` blocks in `.proto` files
- Should already be gone from gRPC removal, but verify
- **Risk:** None

### PR 3.2: Delete proto message definitions + generated code
**Scope:** Remove `proto/prysm/v1alpha1/*.proto` and `proto/prysm/v1alpha1/*.pb.go`
- Verify zero Go files import `eth "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"`
- **Risk:** Low after Phase 2 — all references already migrated
- **Success criteria:** `proto/prysm/v1alpha1/` directory deleted

### PR 3.3: Remove protobuf dependencies from go.mod
**Scope:** Clean up go.mod
```bash
# Remove direct proto deps:
#   google.golang.org/protobuf
#   github.com/golang/protobuf (legacy)
#   google.golang.org/genproto
go mod tidy
bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=deps.bzl%prysm_deps -prune=true
```
- **Risk:** Low — verify no transitive deps still need protobuf
- Note: `google.golang.org/protobuf` may remain as indirect dep (e.g., via OTLP)

### PR 3.4: Delete remaining api/server/structs/ (if any)
**Scope:** Remove any struct types kept from Phase 1
- **Risk:** None

### Phase 3 Gate
- [ ] `proto/prysm/v1alpha1/` directory deleted
- [ ] No `.proto` or `.pb.go` files remain (except engine/v1 if still needed)
- [ ] `google.golang.org/protobuf` removed or indirect-only
- [ ] All unit tests pass
- [ ] E2E tests pass
- [ ] **Estimated removal: ~103,000+ lines (proto definitions + generated Go)**

---

## Summary

| Phase | PRs (est.) | Risk | Lines Removed | Description |
|-------|-----------|------|---------------|-------------|
| 1 | ~7 | Low-Medium | ~8,500 | jsongen on .pb.go, delete structs/ conversions |
| 2 | ~30 | Medium-High | — (replacement) | Native Go structs replace proto types |
| 3 | ~4 | Low | ~103,000+ | Delete .proto + .pb.go (proto definitions + generated Go) |
| **Total** | **~41** | | **~111,500+** | |

## Critical Path

```
Phase 1 (jsongen + delete structs)
         ↓
Phase 2 Wave 1 (leaf types)
         ↓
Phase 2 Wave 2 (1-dep types)
         ↓
Phase 2 Wave 3 (2-dep types)
         ↓
Phase 2 Wave 4 (BeaconBlock hierarchy)
         ↓
Phase 2 Wave 5 (BeaconState)
         ↓
Phase 3 (delete .proto + .pb.go)
```

Phase 1 is independent and can start immediately.
Phase 2 waves are strictly ordered by dependency.
Phase 3 can only start after Phase 2 is 100% complete.

## Open Questions

1. **Package placement:** ~~Put native types in `consensus-types/primitives/`?~~ → **Resolved: `consensus-types/`** (see Decisions below)
2. **PR granularity:** 1 type per PR for Wave 1? Batch small types?
3. **Mixed types during migration:** When a parent type is still proto but a child is native, how to handle? Temporary conversion helpers?
4. **engine/v1 proto:** Keep `proto/engine/v1/` (execution layer types) as proto, or migrate too?
5. **State interface:** `state.BeaconState` interface wraps proto — how does native struct migration interact with the state package?

---

## Decisions (Resolved)

### Package Placement ✅
- **`consensus-types/`** holds all migrated types (not `primitives/`)
- `primitives/` is reserved for low-level types like `Slot`, `Epoch`, `SSZBytes`, `SSZUint64`
- New native types go in appropriate sub-packages under `consensus-types/` (e.g., `consensus-types/blocks/`, or a new `consensus-types/consensus/`)

### PR Granularity ✅
- **Wave 1 leaves: batch** — simple types (2-5 fields, primitives only), no interdependencies. ~2-3 PRs to cover all ~38 types.
- **Wave 2-3: smaller batches** — group by domain (attestation types together, block header types together). ~3-4 PRs each.
- **Wave 4-5: per-fork PRs** — one PR per fork variant for BeaconBlockBody and BeaconState.

### jsongen Timing ✅
- **Phase 1 first:** Run `prysm-jsongen` on existing `.pb.go` files → delete `structs/conversions*.go`
- **Phase 2:** Switch jsongen to run on native structs instead

### Mixed Types During Migration ✅
- **Prefer direct replacement** — swap `ethpb.Checkpoint` → `consensus.Checkpoint` everywhere in one shot per type
- **Mixed types acceptable** if needed during transition — temporary conversion helpers are okay
- Bottom-up wave order minimizes the mixed-type window
