# Proto → Native Go Struct Migration — Investigation

## Problem Statement

Prysm's type system has **three redundant layers**:

1. **Protobuf types** (`proto/prysm/v1alpha1/*.pb.go`) — generated from `.proto` files, used as the canonical internal types across 1,110+ files
2. **API structs** (`api/server/structs/`) — hand-written Go structs with JSON tags for the Beacon API REST layer (~270 struct types, ~10,800 lines)
3. **Conversion functions** (`api/server/structs/conversions*.go`) — `FromConsensus` / `ToConsensus` glue between proto and API structs (~8,500 lines)

Every data type is defined **twice** (proto + structs) and every data flow goes through a **manual conversion layer**. This is:
- **Error-prone** — conversion bugs are a recurring source of issues
- **Maintenance-heavy** — every proto change requires a matching structs + conversion update
- **Performance-wasteful** — unnecessary allocations and copies on every API call

## Goal

Replace protobuf message types with **native Go structs** that directly support:
- **SSZ** encoding via `sszgen` (`github.com/prysmaticlabs/fastssz`, Bazel rule in `tools/ssz.bzl`)
- **JSON** encoding via `prysm-jsongen` (new codegen tool at github.com/syjn99/prysm-jsongen)
- Standard Go patterns (no proto reflection, no `.pb.go` quirks)

## Current Footprint

| Layer | Files | Lines | Types |
|-------|-------|-------|-------|
| Proto definitions (`.proto`) | 20 | ~8,000 | 307 messages |
| Generated Go (`.pb.go`) | 20 | ~95,000 | 307 types |
| API structs | 22 | ~10,800 | ~270 types |
| Conversion functions | 9 | ~8,500 | ~670 functions |
| Proto importers | 1,110+ | — | — |

### Proto message types by file

| Proto file | Messages | Domain |
|------------|----------|--------|
| beacon_state.proto | ~30 | BeaconState (per-fork variants) |
| beacon_block.proto | ~80 | BeaconBlock/Body (per-fork variants) |
| attestation.proto | ~20 | Attestations |
| validator.proto | ~30 | Validator duties & responses |
| beacon_chain.proto | ~15 | Chain queries |
| beacon_core_types.proto | ~15 | Core types (Checkpoint, Fork, etc.) |
| light_client.proto | ~15 | Light client |
| sync_committee.proto | ~10 | Sync committee |
| p2p_messages.proto | ~15 | P2P networking |
| eip_7251.proto | ~10 | EIP-7251 (MaxEB) |
| gloas.proto | ~10 | Gloas upgrade |
| Others | ~50 | Blobs, slasher, powchain, etc. |

## Tools Available

### sszgen
- External tool from `github.com/prysmaticlabs/fastssz`
- Generates `MarshalSSZ`, `UnmarshalSSZ`, `SizeSSZ`, `HashTreeRoot` methods
- Currently reads `go_proto_library` output, but also supports raw `.go` source files via `srcs` attr

### prysm-jsongen (github.com/syjn99/prysm-jsongen)
- Codegen tool that generates `MarshalJSON` / `UnmarshalJSON` with proper hex encoding for byte fields
- Can run on existing `.pb.go` files (Phase 1) OR on native structs (Phase 2)
- Eliminates the need for `api/server/structs/` entirely

## Migration Waves (Dependency Order)

Types must be migrated bottom-up based on their proto dependencies:

### Wave 1 — Pure Leaves (0 proto deps, ~38 types)
Types with no references to other proto message types. Can be migrated independently.

| Type | Parent refs | Fields |
|------|------------|--------|
| SyncAggregate | 21 | 2 (SyncCommitteeBits, SyncCommitteeSignature) |
| Eth1Data | 20 | 3 (DepositRoot, DepositCount, BlockHash) |
| SyncCommittee | 15 | 2 (Pubkeys, AggregatePubkey) |
| Deposit | 13 | 2 (Proof, Data→DepositData) |
| BeaconBlockHeader | 11 | 5 (Slot, ProposerIndex, ParentRoot, StateRoot, BodyRoot) |
| Checkpoint | 9 | 2 (Epoch, Root) |
| Fork | 9 | 3 (PreviousVersion, CurrentVersion, Epoch) |
| Validator | 9 | 8 fields |
| HistoricalSummary | 5 | 2 (BlockSummaryRoot, StateSummaryRoot) |
| VoluntaryExit | 1 | 2 (Epoch, ValidatorIndex) |
| BLSToExecutionChange | 1 | 3 fields |

### Wave 2 — 1-dep types (~42 types)

| Type | Parent refs | Dependency |
|------|------------|------------|
| SignedVoluntaryExit | 11 | VoluntaryExit |
| LightClientHeaderDeneb | 7 | BeaconBlockHeader |
| SignedBLSToExecutionChange | 7 | BLSToExecutionChange |
| AttestationData | 6 | Checkpoint |
| SignedBeaconBlockHeader | 3 | BeaconBlockHeader |

### Wave 3 — 2-dep types (~24 types)

| Type | Parent refs | Dependencies |
|------|------------|--------------|
| ProposerSlashing | 11 | SignedBeaconBlockHeader |
| Attestation | 9 | AttestationData |
| AttestationElectra | 4 | AttestationData |
| IndexedAttestation | — | AttestationData |

### Wave 4+ — BeaconBlock Hierarchy (~63 types)
`BeaconBlockBody → BeaconBlock → SignedBeaconBlock` with per-fork variants.

### Wave 5 — BeaconState (~30 types)
The most complex types. Depend on many Wave 1-3 types.

## Key Constraints

1. **Bazel** — all BUILD.bazel files must be updated; `gazelle` must be run
2. **SSZ compatibility** — encoded output must be bit-identical to proto SSZ
3. **JSON compatibility** — Beacon API JSON must remain spec-compliant
4. **1,110+ importers** — mass refactoring of `ethpb.TypeName` → new package
5. **Mixed types** — during migration, some fields will be native while parent types are still proto

## Related Upstream Issues

- [#15372](https://github.com/OffchainLabs/prysm/issues/15372) — Generate JSON ser/des on methodical generated types
- [#15374](https://github.com/OffchainLabs/prysm/issues/15374) — Simplify and deduplicate type system
- [#15376](https://github.com/OffchainLabs/prysm/issues/15376) — Reduce API type definitions
