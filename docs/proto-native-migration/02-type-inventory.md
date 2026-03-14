# Type Inventory — Proto Messages & Dependencies

## Wave 1 — Pure Leaves (0 proto deps)

These types contain only primitive fields (integers, byte arrays, booleans). No references to other proto message types.

| # | Type | Proto File | Fields | Parent Refs | Struct Exists |
|---|------|-----------|--------|-------------|---------------|
| 1 | SyncAggregate | beacon_block.proto | 2 | 21 | ✅ |
| 2 | Eth1Data | beacon_core_types.proto | 3 | 20 | ✅ |
| 3 | SyncCommittee | sync_committee.proto | 2 | 15 | ✅ |
| 4 | DepositData | beacon_core_types.proto | 4 | — (embedded) | ✅ |
| 5 | Deposit | beacon_core_types.proto | 2 | 13 | ✅ |
| 6 | BeaconBlockHeader | beacon_core_types.proto | 5 | 11 | ✅ |
| 7 | Checkpoint | beacon_core_types.proto | 2 | 9 | ✅ |
| 8 | Fork | beacon_core_types.proto | 3 | 9 | ✅ |
| 9 | Validator | beacon_core_types.proto | 8 | 9 | ✅ |
| 10 | HistoricalSummary | beacon_state.proto | 2 | 5 | ✅ |
| 11 | VoluntaryExit | beacon_core_types.proto | 2 | 1 | ✅ |
| 12 | BLSToExecutionChange | beacon_core_types.proto | 3 | 1 | ✅ |
| 13 | DepositMessage | beacon_core_types.proto | 3 | — | ✅ |
| 14 | ForkData | beacon_core_types.proto | 2 | — | — |
| 15 | SigningData | beacon_core_types.proto | 2 | — | — |
| 16 | PendingDeposit | eip_7251.proto | 5 | — | — |
| 17 | PendingPartialWithdrawal | eip_7251.proto | 3 | — | — |
| 18 | PendingConsolidation | eip_7251.proto | 2 | — | — |
| 19 | Consolidation | eip_7251.proto | 3 | — | — |
| 20 | SignedConsolidation | eip_7251.proto | 2 | — | — |
| 21 | ExecutionLayerWithdrawalRequest | eip_7251.proto | 3 | — | — |
| 22 | PayloadAttestationData | gloas.proto | 2 | — | — |

(+ ~16 more leaf types in p2p_messages, light_client, slasher, etc.)

## Wave 2 — 1-dep Types

| # | Type | Dependency | Parent Refs |
|---|------|-----------|-------------|
| 1 | SignedVoluntaryExit | VoluntaryExit | 11 |
| 2 | LightClientHeaderDeneb | BeaconBlockHeader | 7 |
| 3 | LightClientHeaderCapella | BeaconBlockHeader | — |
| 4 | LightClientHeaderAltair | BeaconBlockHeader | — |
| 5 | SignedBLSToExecutionChange | BLSToExecutionChange | 7 |
| 6 | AttestationData | Checkpoint (×2) | 6 |
| 7 | SignedBeaconBlockHeader | BeaconBlockHeader | 3 |
| 8 | PayloadAttestation | PayloadAttestationData | — |
| 9 | SignedPayloadAttestation | PayloadAttestation | — |

(+ ~33 more 1-dep types)

## Wave 3 — 2-dep Types

| # | Type | Dependencies | Parent Refs |
|---|------|-------------|-------------|
| 1 | ProposerSlashing | SignedBeaconBlockHeader | 11 |
| 2 | Attestation | AttestationData | 9 |
| 3 | AttestationElectra | AttestationData | 4 |
| 4 | IndexedAttestation | AttestationData | — |
| 5 | IndexedAttestationElectra | AttestationData | — |
| 6 | AttesterSlashing | IndexedAttestation | — |
| 7 | AttesterSlashingElectra | IndexedAttestationElectra | — |

(+ ~17 more 2-dep types)

## Wave 4 — BeaconBlockBody (per-fork)

Each BeaconBlockBody variant depends on many Wave 1-3 types:

| Fork | Key Fields (proto deps) |
|------|------------------------|
| Phase0 | Eth1Data, ProposerSlashing[], AttesterSlashing[], Attestation[], Deposit[], VoluntaryExit[] |
| Altair | + SyncAggregate |
| Bellatrix | + ExecutionPayload |
| Capella | + SignedBLSToExecutionChange[], ExecutionPayloadCapella |
| Deneb | + BlobKzgCommitments |
| Electra | + AttestationElectra[], AttesterSlashingElectra[], ExecutionRequests |
| Gloas | + PayloadAttestation[] |

Then: `BeaconBlock = (Slot, ProposerIndex, ParentRoot, StateRoot, Body)`
Then: `SignedBeaconBlock = (Block, Signature)`

**~21 types per fork × 7 forks = ~63 types total** (but many share structure)

## Wave 5 — BeaconState (per-fork)

BeaconState contains references to nearly every other type:
- Validators[], Checkpoint (×3), Fork, Eth1Data, BeaconBlockHeader
- Per-fork additions: SyncCommittee, ExecutionPayloadHeader, etc.

**~7 fork variants, ~30 types total**

## Total Count

| Wave | Types (est.) | Complexity |
|------|-------------|------------|
| 1 | ~38 | Simple (primitives only) |
| 2 | ~42 | Low (1 proto dep) |
| 3 | ~24 | Medium (2 proto deps) |
| 4 | ~63 | High (many deps, per-fork) |
| 5 | ~30 | Very High (depends on everything) |
| **Total** | **~197** | |

Note: Not all 307 proto messages need migration. Some are request/response types for deleted gRPC services, validator API types that only exist for REST, etc.
