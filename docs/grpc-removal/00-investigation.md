# gRPC Removal Investigation

## Upstream Issue
**OffchainLabs/prysm#15346 — "Remove gRPC"**

### Sub-Issues
| Issue | Title | Status | Description |
|-------|-------|--------|-------------|
| #15372 | Generate json ser/des code on methodical generated types | Open | Remove duplicate types for json ser/des |
| #15373 | Simplify engine API code with generated json methods | Open | Depends on methodical json codegen |
| #15374 | Simplify and deduplicate prysm's type system | Open | Remove interface wrappers, replace with generated types with setters |
| #15375 | Merge all api structs and api clients into one location | Open | Consolidate scattered API code |
| #15376 | Reduce API Type definitions in code base | Open | Types defined in 3+ places (api/server/structs, proto/prysm/v1alpha1, api/client/builder/types.go) |
| #16410 | Delete dead code around gRPC attestation APIs | Open | Remove unused ListAttestations, AttestationPool, etc. |

### Comments
- James: "need to also check web ui how to migrate if grpc is removed"
- James: "remove proto/eth/v1 package and update all files based on this"
- syjn99: E2E decoupled from gRPC on fix/REST-only-e2e branch

---

## Current gRPC Footprint

### Files by Area (76 total Go files importing google.golang.org/grpc)

| Directory | Count | Role |
|-----------|-------|------|
| beacon-chain/rpc/ | 36 | gRPC server + v1alpha1 service implementations |
| validator/ | 21 | gRPC client implementations + connection mgmt |
| testing/mock/ | 6 | Generated gRPC mocks |
| api/grpc/ | 5 | Connection provider, utilities |
| proto/prysm/v1alpha1/ | 5 | Generated gRPC stubs (.pb.go) |
| cmd/ | 2 | CLI tools (exit, prysmctl) |
| tools/ | 1 | forkchecker |

### gRPC Server (beacon-chain/rpc/service.go)
- `grpc.NewServer()` with interceptors (recovery, prometheus, opentracing, custom)
- 5 services registered: Node, Health, BeaconChain, Debug, BeaconNodeValidator
- gRPC reflection enabled
- TLS optional

### gRPC Clients
- `api/grpc/grpc_connection_provider.go` — Failover connection manager
- `validator/client/grpc-api/` — 5 client implementations (ValidatorClient, NodeClient, ChainClient, PrysmChainClient, ClientManager)
- `validator/helpers/node_connection.go` — Unified connection wrapper
- `cmd/validator/accounts/exit.go` — Direct grpc.DialContext
- `cmd/prysmctl/p2p/client.go` — Direct grpc.Dial
- `tools/forkchecker/forkchecker.go` — Direct grpc.Dial

### Proto Service Definitions (5 services in proto/prysm/v1alpha1/)
1. `validator.proto` — BeaconNodeValidator (primary validator API)
2. `beacon_chain.proto` — BeaconChain (chain queries)
3. `node.proto` — Node (sync, genesis, version, peers)
4. `health.proto` — Health (log streaming)
5. `debug.proto` — Debug (state/block inspection)

### gRPC Status Code Usage (41 files)
Concentrated in beacon-chain/rpc/prysm/v1alpha1/ (30 files) — these map gRPC status codes (NotFound, InvalidArgument, etc.) to errors.

### proto/eth/v1 Usage (22 files)
Used in: proto/migration/, beacon-chain/blockchain/, testing/util/, events/, etc.

### proto/prysm/v1alpha1 Usage (1,110+ files)
Deeply embedded — but most usage is **message types** (not gRPC stubs). These types are used throughout consensus logic, state management, DB, sync, etc. Removing gRPC does NOT require removing these protobuf message types.

---

## Existing REST Infrastructure

### Beacon-Chain REST Server (already running)
- `api/server/httprest/server.go` — HTTP server with middleware
- `beacon-chain/rpc/endpoints.go` — 60+ REST endpoints registered
- Covers: /eth/v1/, /eth/v2/, /prysm/v1/ prefixes
- All standard Beacon API spec endpoints implemented

### Validator REST Clients (complete alternative)
- `validator/client/beacon-api/` — 90+ files, full REST implementation
- Implements same `iface.ValidatorClient` interface as gRPC client
- All validator duties covered: duties, blocks, attestations, sync committee, etc.

### Factory Pattern (feature flag gated)
- `validator/client/validator-client-factory/` — Switches gRPC ↔ REST
- `validator/client/node-client-factory/` — Same pattern
- `validator/client/beacon-chain-client-factory/` — Same with fallback
- Feature flag: `EnableBeaconRESTApi` (experimental, disabled by default)

### E2E Tests (already decoupled)
- Branch `fix/REST-only-e2e` has removed ALL gRPC from testing/endtoend/
- All 18 evaluators use REST-only `helpers.NewBeaconNodeClient()`
- No gRPC imports remain in testing/ directory

---

## go.mod gRPC Dependencies
```
google.golang.org/grpc v1.71.0
github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0
github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1
```
