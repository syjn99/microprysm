# gRPC Removal Plan — Step-by-Step PRs

## Principles
- Each PR is independently mergeable and does not break tests or functionality
- Order matters — later PRs depend on earlier ones
- Protobuf **message types** (proto/prysm/v1alpha1/*.go) are kept — only gRPC service stubs and clients are removed
- REST alternatives must be verified before removing gRPC counterparts

---

## Phase 1: Dead Code Cleanup (Low Risk)

### PR 1.1: Delete dead gRPC attestation APIs ✅ — [PR #7](https://github.com/syjn99/microprysm/pull/7)
**Scope:** Mirrors upstream #16410
- ~~Remove unused RPCs from `beacon_chain.proto`~~ ✅
- ~~Remove corresponding implementations in `beacon-chain/rpc/prysm/v1alpha1/beacon/attestations.go`~~ ✅
- ~~Remove related test code~~ ✅
- ~~Regenerate proto stubs~~ ✅
- **Risk:** None — these are confirmed unused

### PR 1.2: Delete dead gRPC beacon chain APIs — [PR #10](https://github.com/syjn99/microprysm/pull/10)
**Scope:** Audit remaining BeaconChain service RPCs for unused ones
- ~~ListBlocks, ListBlocksElectra~~ ✅ (removed in PR #10)
- ~~GetChainHead~~ ✅ (removed in PR #10)
- ~~GetValidatorParticipation~~ ✅ (removed in PR #10)
- Remaining candidate: GetWeakSubjectivity
- Check if each has callers (gRPC client code, E2E, CLI tools)
- Remove those with zero callers
- **Risk:** Low — requires careful audit

### PR 1.3: Delete dead gRPC debug APIs ✅ — [PR #8](https://github.com/syjn99/microprysm/pull/8)
**Scope:** Audit Debug service RPCs
- ~~Check: GetBeaconState, GetBlock, SetLoggingLevel, ListPeers, GetPeer~~ ✅
- ~~Remove unused RPCs and implementations~~ ✅ (entire `debug/` directory deleted)
- ~~Removed `service Debug` block from `debug.proto`~~ ✅
- ~~Unregistered debug service from `rpc/service.go`~~ ✅
- **Risk:** Low

---

## Phase 2: Remove gRPC from CLI Tools (Low Risk)

### PR 2.1: Remove gRPC from tools/forkchecker ✅ — [PR #5](https://github.com/syjn99/microprysm/pull/5)
**Scope:** `tools/forkchecker/forkchecker.go`
- This tool uses `grpc.Dial()` to connect to a beacon node
- Option A: Rewrite to use REST API
- Option B: Delete the tool if no longer needed (likely — it's a debugging tool)
- **Risk:** None — standalone tool

### PR 2.2: Remove gRPC from cmd/prysmctl/p2p ✅ — [PR #6](https://github.com/syjn99/microprysm/pull/6)
**Scope:** `cmd/prysmctl/p2p/client.go`
- ~~Uses `grpc.Dial()` for p2p debugging~~ ✅
- ~~Rewritten to use REST debug endpoints~~ ✅
- **Risk:** Low — CLI debugging tool

### PR 2.3: Remove gRPC from cmd/validator/accounts/exit ✅ — [PR #9](https://github.com/syjn99/microprysm/pull/9)
**Scope:** `cmd/validator/accounts/exit.go`
- ~~Uses `grpc.DialContext()` for voluntary exit~~
- Rewritten to use REST API (`/eth/v1/beacon/genesis` for genesis info, REST connection provider for exit submission)
- **Risk:** Low — single endpoint migration

---

## Phase 3: Make REST Default for Validator Client (Medium Risk)

### PR 3.1: Make EnableBeaconRESTApi default to true ✅
**Scope:** `config/features/config.go`, `config/features/flags.go`
- ~~Flip the feature flag default from false to true~~ ✅
- ~~All validator clients will now use REST by default~~ ✅
- gRPC still available as fallback via `--enable-beacon-rest-api=false`
- **Risk:** Medium — needs thorough testing. Run E2E tests.
- **Prerequisite:** Verify all validator duties work via REST

### PR 3.2: Remove gRPC from factories ✅
**Scope:** `validator/client/*-factory/`
- Remove the feature flag check entirely
- Factories always return REST implementations
- Remove gRPC fallback wrappers (NewNodeClientWithFallback, NewBeaconApiChainClientWithFallback)
- **Risk:** Medium — REST must handle all cases

### PR 3.3: Delete validator gRPC client implementations ✅
**Scope:** `validator/client/grpc-api/` (entire directory)
- ~~Delete: grpc_validator_client.go, grpc_node_client.go, grpc_beacon_chain_client.go, grpc_prysm_beacon_chain_client.go, grpc_client_manager.go~~ ✅
- ~~Delete corresponding test files~~ ✅
- **Risk:** Low after PR 3.2 (no remaining references)

### PR 3.4: Remove gRPC from validator connection setup ✅
**Scope:** `validator/helpers/node_connection.go`, `validator/client/service.go`
- ~~Remove `WithGRPC()` option and `GetGrpcClientConn()` / `GetGrpcConnectionProvider()` methods~~ ✅
- ~~Remove gRPC dial options construction from service.go (`ConstructDialOptions()`)~~ ✅
- ~~Remove gRPC-related CLI flags from validator~~ ✅
- **Risk:** Low after PR 3.3

### PR 3.5: Remove gRPC from validator/rpc (validator HTTP server) ✅ — [PR #14](https://github.com/syjn99/microprysm/pull/14)
**Scope:** `validator/rpc/beacon.go`, `validator/rpc/intercepter.go`
- ~~Remove gRPC middleware imports and interceptor code~~ ✅
- ~~Simplify beacon node connection to REST-only~~ ✅
- **Risk:** Low

### PR 3.6: Remove gRPC from validator/accounts ✅
**Scope:** `validator/accounts/cli_manager.go`, `validator/accounts/cli_options.go`
- ~~Remove gRPC dial options configuration~~ ✅
- **Risk:** Low

---

## Phase 4: Remove gRPC Server from Beacon Chain (High Risk)

### PR 4.1: Remove gRPC service registrations
**Scope:** `beacon-chain/rpc/service.go`
- Remove `grpc.NewServer()` creation
- Remove all `RegisterXxxServer()` calls
- Remove gRPC listener, interceptors, TLS setup
- Remove `grpc.Serve()` and `GracefulStop()`
- Keep the HTTP REST server running (already separate)
- **Risk:** High — major architectural change. Must verify no internal code calls gRPC services directly.

### PR 4.2: Delete v1alpha1 gRPC validator service implementation ✅
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/validator/` (entire directory, ~60 files)
- Extracted proposer logic into new `beacon-chain/rpc/proposer/` package
- Updated call sites: `eth/validator/`, `eth/beacon/`, `endpoints.go`, `service.go`
- Removed `RegisterBeaconNodeValidatorServer` gRPC registration
- Deleted entire `v1alpha1/validator/` directory
- **Risk:** Medium — verify REST endpoints cover all functionality

### ~~PR 4.3: Delete v1alpha1 gRPC beacon service implementation~~ ✅
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/beacon/` (~6 files)
- REST equivalents in `beacon-chain/rpc/eth/beacon/`
- **Risk:** Medium

### PR 4.4: Delete v1alpha1 gRPC node service implementation ✅
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/node/` (~2 files)
- REST equivalents in `beacon-chain/rpc/eth/node/`
- **Risk:** Low

### PR 4.5: Delete v1alpha1 gRPC debug service implementation
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/debug/` (~4 files)
- REST equivalents in `beacon-chain/rpc/eth/debug/`
- **Risk:** Low

### PR 4.6: Delete Health gRPC service (StreamBeaconLogs) ✅
**Scope:** Health service in node server
- REST equivalent: SSE events endpoint `/eth/v1/events`
- Verify log streaming works via SSE or remove if not needed
- **Risk:** Low

---

## Phase 5: Remove gRPC Infrastructure (Low Risk after Phase 4)

### PR 5.1: Delete api/grpc/ package ✅ — [PR #24](https://github.com/syjn99/microprysm/pull/24)
**Scope:** `api/grpc/` (5 files)
- GrpcConnectionProvider, grpcutils, mock
- No remaining callers after Phase 3+4
- **Risk:** None

### PR 5.2: Delete testing/mock/ gRPC mocks ✅ — [PR #25](https://github.com/syjn99/microprysm/pull/25)
**Scope:** `testing/mock/` (6 files)
- beacon_service_mock.go, beacon_validator_client_mock.go, etc.
- Verify no tests reference these mocks
- **Risk:** Low

### PR 5.3: Remove gRPC service definitions from proto files ✅ — [PR #28](https://github.com/syjn99/microprysm/pull/28)
**Scope:** `proto/prysm/v1alpha1/*.proto`
- Remove `service` blocks from validator.proto, beacon_chain.proto, node.proto, health.proto, debug.proto
- Keep `message` definitions (still used everywhere)
- Regenerate .pb.go files (will remove _grpc.pb.go stubs)
- **Risk:** Low — message types preserved

### PR 5.4: Remove gRPC from beacon-chain/rpc/core/ ✅ — [PR #26](https://github.com/syjn99/microprysm/pull/26)
**Scope:** `beacon-chain/rpc/core/errors.go`, `beacon-chain/rpc/core/duties.go`
- Replace gRPC status codes with standard errors or HTTP-compatible error types
- **Risk:** Low

### PR 5.5: Remove gRPC from beacon-chain/rpc/eth/helpers/ ✅ — merged with PR #26
**Scope:** `beacon-chain/rpc/eth/helpers/error_handling.go`
- Replace any gRPC status code conversion with HTTP status helpers
- **Risk:** Low

### PR 5.6: Remove EnableBeaconRESTApi feature flag ✅ — [PR #29](https://github.com/syjn99/microprysm/pull/29)
**Scope:** `config/features/`, `testing/endtoend/types/`, `testing/endtoend/`
- Flag no longer meaningful — REST is the only option
- Clean up flag definition and config field
- Remove `WithValidatorRESTApi()` E2E config option and `UseBeaconRestApi` field
- Remove dedicated `TestEndToEnd_MinimalConfig_ValidatorRESTApi` test variants (redundant)
- **Risk:** None

---

## Phase 6: Remove proto/eth/v1 Package (Medium Risk)

### PR 6.1: Migrate proto/migration/ away from ethv1 ✅ — [PR #32](https://github.com/syjn99/microprysm/pull/32)
**Scope:** `proto/migration/v1alpha1_to_v1.go` (4 files)
- These convert between v1alpha1 and v1 types
- Replace with direct conversions to api/server/structs types
- **Risk:** Medium — used by multiple packages

### PR 6.2: Remove ethv1 from beacon-chain/ ✅ — [PR #34](https://github.com/syjn99/microprysm/pull/34)
**Scope:** 8 files in beacon-chain/ importing ethv1
- blockchain/receive_block.go, head.go
- rpc/eth/events/events.go
- rpc/eth/helpers/, rpc/eth/node/
- p2p/, p2p/peers/
- **Risk:** Medium — requires type changes

### PR 6.3: Remove ethv1 from testing/util/ ✅ — [PR #33](https://github.com/syjn99/microprysm/pull/33)
**Scope:** 4 files in testing/util/
- attestation.go, block.go
- Replace ethv1 types with v1alpha1 or structs types
- **Risk:** Low

### PR 6.4: Delete proto/eth/v1/ directory ✅ — [PR #35](https://github.com/syjn99/microprysm/pull/35)
**Scope:** Entire `proto/eth/v1/` directory
- After all references removed
- **Risk:** None

---

## Phase 6.5: Clean Up Remaining proto/eth/v1 Test References

### PR 6.5: Remove proto/eth/v1 from test files
**Scope:** Test files still importing ethv1 types
- `beacon-chain/p2p/peers/status_test.go` — ethv1 peer status types
- `beacon-chain/p2p/connection_gater_test.go` — ethv1 peer types
- Remove remaining `.pb.go` / `.ssz.go` files if still present after Phase 6.4
- **Risk:** None — test-only changes

---

## Phase 7: Clean Up go.mod and Bazel Dependencies

### PR 7.1: Remove gRPC dependencies from go.mod + deps.bzl
**Scope:** `go.mod`, `go.sum`, `deps.bzl`, `BUILD.bazel` files

#### Step 1: Remove gRPC imports from Go source
- Verify no `.go` files import `google.golang.org/grpc` (should be zero after Phase 6)
- Check for transitive imports via other packages

#### Step 2: Clean up BUILD.bazel gRPC references
- `proto/prysm/v1alpha1/BUILD.bazel` — remove `@org_golang_google_grpc` deps
- `proto/prysm/v1alpha1/validator-client/BUILD.bazel` — remove `@org_golang_google_grpc` deps
- `beacon-chain/rpc/proposer/BUILD.bazel` — remove `@org_golang_google_grpc` deps (note: has `# gazelle:ignore`)
- Root `BUILD.bazel` — remove `grpc_proto_compiler` and `cast_grpc_proto_compiler` aliases
- Remove `go_cast_grpc` compiler references from proto BUILD targets

#### Step 3: Remove Go module dependencies
```bash
# Edit go.mod to remove direct gRPC deps:
#   google.golang.org/grpc
#   github.com/grpc-ecosystem/go-grpc-middleware
#   github.com/grpc-ecosystem/go-grpc-prometheus
#   go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
#   github.com/grpc-ecosystem/grpc-gateway/v2

# Clean up transitive deps
go mod tidy

# Update Bazel deps (see DEPENDENCIES.md)
bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=deps.bzl%prysm_deps -prune=true
```

#### Step 4: Verify
```bash
# No gRPC imports remain
grep -r "google.golang.org/grpc" --include="*.go" -l | grep -v vendor/

# Build passes
bazel build //...

# Tests pass
bazel test //...
```

- **Risk:** Low — final cleanup. Main risk is transitive deps that still need gRPC indirectly.

---

## Summary

| Phase | PRs | Risk | Description |
|-------|-----|------|-------------|
| 1 | 3 | Low | Dead code cleanup |
| 2 | 3 | Low | CLI tools migration |
| 3 | 6 | Medium | Validator client → REST-only |
| 4 | 6 | High | Beacon chain gRPC server removal |
| 5 | 6 | Low | Infrastructure cleanup |
| 6 | 4 | Medium | proto/eth/v1 removal |
| 6.5 | 1 | None | proto/eth/v1 test reference cleanup |
| 7 | 1 | Low | go.mod + deps.bzl cleanup |
| **Total** | **30** | | |

### Critical Path
```
Phase 1 (dead code) → Phase 2 (CLI tools) → Phase 3 (validator REST-only)
                                                      ↓
                                              Phase 4 (beacon gRPC server)
                                                      ↓
                                              Phase 5 (infrastructure)
                                                      ↓
                                              Phase 6 (proto/eth/v1)
                                                      ↓
                                              Phase 7 (go.mod)
```

Phases 1 and 2 can proceed in parallel.
Phase 3 must complete before Phase 4.
Phases 5 and 6 can partially overlap after Phase 4.

### PR 5.7: Remove gRPC status codes from proposer and validator/client ✅ — [PR #31](https://github.com/syjn99/microprysm/pull/31)
**Scope:** Final gRPC cleanup in non-proto Go files
- beacon-chain/rpc/proposer/ (5 files) — `grpc/codes`, `grpc/status` → `fmt.Errorf`/`errors.New`
- validator/client/ (3 files) — `grpc/codes`, `grpc/status`, `grpc/metadata` → standard errors
- **Risk:** Low — error type changes only, no behavioral changes
- **Note:** Also closes PR #27 (Phase 5.5 original attempt was wrong approach)
