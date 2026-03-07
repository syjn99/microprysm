# gRPC Removal Plan — Step-by-Step PRs

## Principles
- Each PR is independently mergeable and does not break tests or functionality
- Order matters — later PRs depend on earlier ones
- Protobuf **message types** (proto/prysm/v1alpha1/*.go) are kept — only gRPC service stubs and clients are removed
- REST alternatives must be verified before removing gRPC counterparts

---

## Phase 1: Dead Code Cleanup (Low Risk)

### PR 1.1: Delete dead gRPC attestation APIs ✅ Complete ([PR #7](https://github.com/syjn99/microprysm/pull/7))
**Scope:** Mirrors upstream #16410
- Remove unused RPCs from `beacon_chain.proto`: ListAttestations, ListAttestationsElectra, ListIndexedAttestations, ListIndexedAttestationsElectra, AttestationPool, AttestationPoolElectra
- Remove corresponding implementations in `beacon-chain/rpc/prysm/v1alpha1/beacon/attestations.go`
- Remove related test code
- Regenerate proto stubs
- **Risk:** None — these are confirmed unused

### PR 1.2: Delete dead gRPC beacon chain APIs
**Scope:** Audit remaining BeaconChain service RPCs for unused ones
- Candidates: ListBlocks, ListBlocksElectra, GetChainHead, GetValidatorParticipation, GetWeakSubjectivity
- Check if each has callers (gRPC client code, E2E, CLI tools)
- Remove those with zero callers
- **Risk:** Low — requires careful audit

### PR 1.3: Delete dead gRPC debug APIs
**Scope:** Audit Debug service RPCs
- Check: GetBeaconState, GetBlock, SetLoggingLevel, ListPeers, GetPeer
- Remove unused RPCs and implementations
- **Risk:** Low

---

## Phase 2: Remove gRPC from CLI Tools (Low Risk)

### PR 2.1: Remove gRPC from tools/forkchecker
**Scope:** `tools/forkchecker/forkchecker.go`
- This tool uses `grpc.Dial()` to connect to a beacon node
- Option A: Rewrite to use REST API
- Option B: Delete the tool if no longer needed (likely — it's a debugging tool)
- **Risk:** None — standalone tool

### PR 2.2: Remove gRPC from cmd/prysmctl/p2p
**Scope:** `cmd/prysmctl/p2p/client.go`
- Uses `grpc.Dial()` for p2p debugging
- Rewrite to use REST debug endpoints or remove if unused
- **Risk:** Low — CLI debugging tool

### PR 2.3: Remove gRPC from cmd/validator/accounts/exit
**Scope:** `cmd/validator/accounts/exit.go`
- Uses `grpc.DialContext()` for voluntary exit
- Rewrite to use REST API (`/eth/v1/beacon/pool/voluntary_exits`)
- **Risk:** Low — single endpoint migration

---

## Phase 3: Make REST Default for Validator Client (Medium Risk)

### PR 3.1: Make EnableBeaconRESTApi default to true
**Scope:** `config/features/config.go`, `config/features/flags.go`
- Flip the feature flag default from false to true
- All validator clients will now use REST by default
- gRPC still available as fallback
- **Risk:** Medium — needs thorough testing. Run E2E tests.
- **Prerequisite:** Verify all validator duties work via REST

### PR 3.2: Remove gRPC fallback from factories
**Scope:** `validator/client/*-factory/`
- Remove the feature flag check entirely
- Factories always return REST implementations
- Remove gRPC fallback wrappers (NewNodeClientWithFallback, NewBeaconApiChainClientWithFallback)
- **Risk:** Medium — REST must handle all cases

### PR 3.3: Delete validator gRPC client implementations
**Scope:** `validator/client/grpc-api/` (entire directory)
- Delete: grpc_validator_client.go, grpc_node_client.go, grpc_beacon_chain_client.go, grpc_prysm_beacon_chain_client.go, grpc_client_manager.go
- Delete corresponding test files
- **Risk:** Low after PR 3.2 (no remaining references)

### PR 3.4: Remove gRPC from validator connection setup
**Scope:** `validator/helpers/node_connection.go`, `validator/client/service.go`
- Remove `WithGRPC()` option and `GetGrpcClientConn()` / `GetGrpcConnectionProvider()` methods
- Remove gRPC dial options construction from service.go (`ConstructDialOptions()`)
- Remove gRPC-related CLI flags from validator
- **Risk:** Low after PR 3.3

### PR 3.5: Remove gRPC from validator/rpc (validator HTTP server)
**Scope:** `validator/rpc/beacon.go`, `validator/rpc/intercepter.go`
- Remove gRPC middleware imports and interceptor code
- Simplify beacon node connection to REST-only
- **Risk:** Low

### PR 3.6: Remove gRPC from validator/accounts
**Scope:** `validator/accounts/cli_manager.go`, `validator/accounts/cli_options.go`
- Remove gRPC dial options configuration
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

### PR 4.2: Delete v1alpha1 gRPC validator service implementation
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/validator/` (entire directory, ~14 files)
- The REST endpoints in `beacon-chain/rpc/eth/validator/` already cover all duties
- Both REST and gRPC handlers delegate to the same `core/` logic
- **Risk:** Medium — verify REST endpoints cover all functionality

### PR 4.3: Delete v1alpha1 gRPC beacon service implementation
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/beacon/` (~6 files)
- REST equivalents in `beacon-chain/rpc/eth/beacon/`
- **Risk:** Medium

### PR 4.4: Delete v1alpha1 gRPC node service implementation
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/node/` (~2 files)
- REST equivalents in `beacon-chain/rpc/eth/node/`
- **Risk:** Low

### PR 4.5: Delete v1alpha1 gRPC debug service implementation
**Scope:** `beacon-chain/rpc/prysm/v1alpha1/debug/` (~4 files)
- REST equivalents in `beacon-chain/rpc/eth/debug/`
- **Risk:** Low

### PR 4.6: Delete Health gRPC service (StreamBeaconLogs)
**Scope:** Health service in node server
- REST equivalent: SSE events endpoint `/eth/v1/events`
- Verify log streaming works via SSE or remove if not needed
- **Risk:** Low

---

## Phase 5: Remove gRPC Infrastructure (Low Risk after Phase 4)

### PR 5.1: Delete api/grpc/ package
**Scope:** `api/grpc/` (5 files)
- GrpcConnectionProvider, grpcutils, mock
- No remaining callers after Phase 3+4
- **Risk:** None

### PR 5.2: Delete testing/mock/ gRPC mocks
**Scope:** `testing/mock/` (6 files)
- beacon_service_mock.go, beacon_validator_client_mock.go, etc.
- Verify no tests reference these mocks
- **Risk:** Low

### PR 5.3: Remove gRPC service definitions from proto files
**Scope:** `proto/prysm/v1alpha1/*.proto`
- Remove `service` blocks from validator.proto, beacon_chain.proto, node.proto, health.proto, debug.proto
- Keep `message` definitions (still used everywhere)
- Regenerate .pb.go files (will remove _grpc.pb.go stubs)
- **Risk:** Low — message types preserved

### PR 5.4: Remove gRPC from beacon-chain/rpc/core/
**Scope:** `beacon-chain/rpc/core/errors.go`, `beacon-chain/rpc/core/duties.go`
- Replace gRPC status codes with standard errors or HTTP-compatible error types
- **Risk:** Low

### PR 5.5: Remove gRPC from beacon-chain/rpc/eth/helpers/
**Scope:** `beacon-chain/rpc/eth/helpers/error_handling.go`
- Replace any gRPC status code conversion with HTTP status helpers
- **Risk:** Low

### PR 5.6: Remove EnableBeaconRESTApi feature flag
**Scope:** `config/features/`
- Flag no longer meaningful — REST is the only option
- Clean up flag definition and config field
- **Risk:** None

---

## Phase 6: Remove proto/eth/v1 Package (Medium Risk)

### PR 6.1: Migrate proto/migration/ away from ethv1
**Scope:** `proto/migration/v1alpha1_to_v1.go` (4 files)
- These convert between v1alpha1 and v1 types
- Replace with direct conversions to api/server/structs types
- **Risk:** Medium — used by multiple packages

### PR 6.2: Remove ethv1 from beacon-chain/
**Scope:** 8 files in beacon-chain/ importing ethv1
- blockchain/receive_block.go, head.go
- rpc/eth/events/events.go
- rpc/eth/helpers/, rpc/eth/node/
- p2p/, p2p/peers/
- **Risk:** Medium — requires type changes

### PR 6.3: Remove ethv1 from testing/util/
**Scope:** 4 files in testing/util/
- attestation.go, block.go
- Replace ethv1 types with v1alpha1 or structs types
- **Risk:** Low

### PR 6.4: Delete proto/eth/v1/ directory
**Scope:** Entire `proto/eth/v1/` directory
- After all references removed
- **Risk:** None

---

## Phase 7: Clean Up go.mod Dependencies

### PR 7.1: Remove gRPC dependencies from go.mod
**Scope:** `go.mod`
- Remove: google.golang.org/grpc
- Remove: github.com/grpc-ecosystem/go-grpc-middleware
- Remove: github.com/grpc-ecosystem/go-grpc-prometheus
- Remove: go.opentelemetry.io/contrib/instrumentation/.../otelgrpc
- Remove: github.com/grpc-ecosystem/grpc-gateway/v2
- Run `go mod tidy`
- **Risk:** None — final cleanup

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
| 7 | 1 | None | go.mod cleanup |
| **Total** | **29** | | |

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
