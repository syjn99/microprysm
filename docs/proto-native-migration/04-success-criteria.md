# Success Criteria

## Overall Success

The migration is complete when:

1. **Zero `ethpb.*` message type references** in non-test Go code (except `proto/engine/v1` if retained)
2. **Zero `api/server/structs/conversions*.go`** files exist
3. **Zero `.proto` message definitions** in `proto/prysm/v1alpha1/` (service blocks already removed)
4. **Zero `.pb.go` files** in `proto/prysm/v1alpha1/`
5. **All E2E tests pass** on the final state
6. **SSZ encoding is bit-identical** to pre-migration for all types (verified by round-trip tests)
7. **JSON encoding is spec-compliant** for all Beacon API responses
8. **Existing databases are readable** without migration (SSZ schema unchanged)
9. **P2P interoperability maintained** (same SSZ on the wire)

## Per-Phase Success Criteria

### Phase 1: jsongen on .pb.go → Delete structs/

| # | Criterion | Verification |
|---|----------|-------------|
| 1 | `prysm-jsongen` generates correct JSON for all proto types | Diff test vs current `structs.XxxFromConsensus()` output |
| 2 | All REST API endpoints return identical JSON | API response comparison test |
| 3 | `api/server/structs/conversions*.go` deleted (~8,500 lines) | `find . -name 'conversions*.go' -path '*/structs/*'` returns empty |
| 4 | No `FromConsensus`/`ToConsensus` calls remain | `grep -r 'FromConsensus\|ToConsensus' --include='*.go'` returns empty |
| 5 | All unit tests pass | `bazel test //...` |
| 6 | E2E tests pass | CI green |

### Phase 2: Native Go Structs

| # | Criterion | Verification |
|---|----------|-------------|
| 1 | Native struct SSZ == proto SSZ (per type) | `TestSSZRoundTrip_NativeVsProto` for each type |
| 2 | Native struct JSON == spec JSON (per type) | `TestJSONCompat` for each type |
| 3 | HashTreeRoot identical | `TestHTR_NativeVsProto` for each type |
| 4 | No `ethpb.*` message type imports remain | `grep -r 'ethpb\.\|eth ".*v1alpha1"' --include='*.go'` |
| 5 | DB reads work with native types | `TestDBRead_NativeTypes` |
| 6 | P2P gossip encoding unchanged | `TestP2PGossip_NativeTypes` |
| 7 | Block processing unchanged | E2E block proposal + validation |
| 8 | State transitions unchanged | E2E epoch transitions |
| 9 | All unit tests pass | `bazel test //...` |
| 10 | E2E tests pass | CI green |

### Phase 3: Delete .proto + .pb.go

| # | Criterion | Verification |
|---|----------|-------------|
| 1 | `proto/prysm/v1alpha1/*.proto` deleted | Directory empty or removed |
| 2 | `proto/prysm/v1alpha1/*.pb.go` deleted | No generated files remain |
| 3 | `google.golang.org/protobuf` removed or indirect-only | `go mod graph \| grep protobuf` |
| 4 | All unit tests pass | `bazel test //...` |
| 5 | E2E tests pass | CI green |

## Net Impact (Estimated)

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| Proto definitions (`.proto`) | ~8,000 lines | 0 | **-8,000** |
| Generated Go (`.pb.go`) | ~95,000 lines | 0 | **-95,000** |
| API structs + conversions | ~10,800 lines | ~2,300 (types only, no conversions) | **-8,500** |
| Total code | — | — | **~-111,500 lines** |
| Type definitions | 3 layers | 1 layer | **-2 layers** |
| Conversion functions | ~670 | 0 | **-670 functions** |
| External deps | protobuf + grpc | — | **-protobuf** |

## Tracking

Progress will be tracked in this document and via GitHub PR labels:
- `proto-migration/phase-1` — Phase 1 PRs
- `proto-migration/phase-2` — Phase 2 PRs  
- `proto-migration/phase-3` — Phase 3 PRs

Each merged PR updates this doc with ✅ status.
