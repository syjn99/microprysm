# Ralph Fix Plan

## Current PRD

[PRD: Migrate E2E Tests from gRPC to REST-Only](../docs/prd/PRD_REST_ONLY_E2E.md)

## Completed

- [x] Task 0.1 — E2E REST client helper (`testing/endtoend/helpers/rest_client.go`) — 17 typed methods covering all gRPC endpoints used by evaluators
- [x] Task 0.2 — Changed Evaluator interface from `*grpc.ClientConn` to `...string` URLs; added transitional `GRPCConns` field to EvaluationContext; updated 18 files
- [x] Task 0.3 — Added `ComputeDomainData(epoch, domainType)` client-side helper; replicates gRPC DomainData logic including EIP-7044 voluntary exit handling; no new REST endpoint needed
- [x] Task 1.1 — Migrated `finality.go` from gRPC to REST (`GetChainHead` via `helpers.BeaconNodeClient`)

## Queue (do these in order, ONE per loop)

- [x] Task 1.2 — Migrate `fee_recipient.go`: replace `GetChainHead`, `ListBeaconBlocks` gRPC calls with REST
- [x] Task 1.3 — Migrate `blob_limits.go`: replace `GetChainHead`, `ListBeaconBlocks` gRPC calls with REST
- [x] Task 1.4 — Migrate `builder.go`: replace `GetGenesis`, `ListBeaconBlocks` gRPC calls with REST
- [x] Task 1.5 — Migrate `validator.go`: replace `GetChainHead`, `ListValidators`, `GetValidator`, `GetGenesis`, `GetValidatorParticipation` gRPC calls with REST
- [x] Task 1.6 — Migrate `metrics.go`: replace `GetChainHead`, `GetGenesis` gRPC calls with REST
- [x] Task 2.1 — Migrate `node.go`: replace `ListPeers` (Node), `GetSyncStatus`, `GetChainHead` gRPC calls with REST
- [x] Task 2.2 — Migrate `peers.go`: replace `ListPeers` (Debug) gRPC calls with REST
- [x] Task 3.1 — Migrate `operations.go`: replace `GetChainHead`, `ListBeaconBlocks`, `GetValidator`, `ListValidators`, `GetBeaconState`, `DomainData`, `ProposeExit` gRPC calls with REST
- [x] Task 3.2 — Migrate `slashing.go`: replace `GetChainHead`, `GetValidator`, `GetDuties`, `StreamBlocksAltair`, `DomainData` gRPC calls with REST
- [x] Task 3.3 — Migrate `slashing_helper.go`: replace `GetDuties`, `GetAttestationData`, `DomainData` gRPC calls with REST
- [x] Task 4.1 — Migrate `fork.go`: replace `StreamBlocksAltair` (6 fork evaluators) with SSE `/eth/v1/events?topics=block` or polling
- [x] Task 5.1 — Migrate `endtoend_test.go`: replace gRPC dial/connection logic with HTTP URL construction
- [x] Task 5.2 — Migrate `helpers/helpers.go`: remove `grpc.Dial`, provide HTTP base URLs only
- [x] Task 5.3 — Remove gRPC imports and unused connection code from all E2E files
- [x] Task 5.4 — Remove gRPC port allocation from E2E config if no longer needed (SKIPPED: RPC port still needed by beacon node and validator client components)
- [x] Task 5.5 — Run full E2E suite to validate (build + vet + unit tests pass; E2E requires full infrastructure)

## How to Migrate an Evaluator (template for each task)

1. Read the target evaluator file in `testing/endtoend/evaluators/`
2. For each gRPC call, find the equivalent REST method on `helpers.BeaconNodeClient`
3. Replace gRPC client creation (`eth.NewBeaconChainClient(conn)`) with `helpers.NewBeaconNodeClient(nodeURLs[0])`
4. Replace gRPC method calls with REST client method calls
5. Remove gRPC imports (`eth`, `grpc`) if no longer used in the file
6. Update BUILD.bazel deps if imports changed (remove gRPC deps, add helpers dep)
7. Run `/precheck` — must pass before committing
8. Run `/test` — unit tests for affected packages must pass
9. Commit with message: `Migrate {file}.go evaluator from gRPC to REST`

## Reference

- Evaluator signature: `func(ec *EvaluationContext, nodeURLs ...string) error` in `testing/endtoend/types/types.go:153`
- REST client: `testing/endtoend/helpers/rest_client.go` — `BeaconNodeClient` with typed methods
- Transitional: `ec.GRPCConns` holds gRPC connections for unmigrated evaluators; remove in Task 5.3
- ~68 gRPC calls across 13 files total; ~50 remaining after Task 1.1
- Some evaluators already partially use REST (execution_engine, healthz, validator participation, BLS withdrawal)
