# Proto Field Name vs Beacon API Spec — Investigation

## Background

Prysm's protobuf definitions use field names that diverge from the [Ethereum Beacon API spec](https://ethereum.github.io/beacon-APIs/) and [consensus-specs](https://github.com/ethereum/consensus-specs). This causes JSON serialization mismatches when using codegen tools like `prysm-jsongen` that derive JSON field names from proto field names.

These discrepancies were discovered during `prysm-jsongen` oracle comparison tests ([CI run](https://github.com/syjn99/prysm-jsongen/actions/runs/22812318921/job/66171261893)).

Currently, Prysm works around these mismatches with hand-written `MarshalJSON`/`UnmarshalJSON` methods in `api/server/structs`. Aligning proto field names to spec eliminates the need for these workarounds and is a prerequisite for the proto → native Go struct migration.

## Methodology

Compared all proto message field names against Beacon API spec YAML definitions (`ethereum/beacon-APIs`) and consensus-specs Python naming conventions.

Source protos:
- `proto/prysm/v1alpha1/*.proto`
- `proto/engine/v1/*.proto`

## Discrepancies Found: 22 fields across 6 groups

### Group 1: Signed wrapper `block`/`header`/`exit` → `message` (10 fields)

The Beacon API spec consistently uses `message` for the inner object of all `Signed*` wrapper types. Prysm uses the specific type name instead.

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `SignedBeaconBlock` | `block` | `message` | 1 |
| `SignedBeaconBlockAltair` | `block` | `message` | 1 |
| `SignedBeaconBlockBellatrix` | `block` | `message` | 1 |
| `SignedBeaconBlockCapella` | `block` | `message` | 1 |
| `SignedBeaconBlockDeneb` | `block` | `message` | 1 |
| `SignedBeaconBlockElectra` | `block` | `message` | 1 |
| `SignedBeaconBlockFulu` | `block` | `message` | 1 |
| `SignedBeaconBlockGloas` | `block` | `message` | 1 |
| `SignedBeaconBlockHeader` | `header` | `message` | 1 |
| `SignedVoluntaryExit` | `exit` | `message` | 1 |

**Note:** `SignedAggregateAttestationAndProof`, `SignedContributionAndProof`, `SignedBLSToExecutionChange` already use `message` correctly.

**Go impact:** ~3,100 references to `.Block`, plus `.Header` and `.Exit` references.

### Group 2: `AttestationData.committee_index` → `index` (1 field)

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `AttestationData` | `committee_index` | `index` | 2 |

**Go impact:** ~380 references to `.CommitteeIndex` on AttestationData.

### Group 3: `block_root` → `beacon_block_root` (2 fields)

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `SyncCommitteeMessage` | `block_root` | `beacon_block_root` | 2 |
| `SyncCommitteeContribution` | `block_root` | `beacon_block_root` | 2 |

**Go impact:** ~294 references to `.BlockRoot` (but not all are these two types — selective rename needed).

### Group 4: `SingleAttestation.committee_id` → `committee_index` (1 field)

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `SingleAttestation` | `committee_id` | `committee_index` | 1 |

**Go impact:** ~29 references.

### Group 5: `ProposerSlashing.header_1/2` → `signed_header_1/2` (2 fields)

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `ProposerSlashing` | `header_1` | `signed_header_1` | 2 |
| `ProposerSlashing` | `header_2` | `signed_header_2` | 3 |

**Go impact:** ~82 references.

### Group 6: `public_key` → `pubkey` (2 fields)

| Proto Message | Proto Field | Spec Field | Proto Field # |
|---|---|---|---|
| `Validator` | `public_key` | `pubkey` | 1 |
| `Deposit.Data` | `public_key` | `pubkey` | 1 |

**Note:** Proto already has `(ethereum.eth.ext.spec_name) = "pubkey"` annotation for SSZ. JSON field name still derives from proto field name `public_key`.

**Go impact:** ~265 references (non-test, non-generated).

## Types Already Correct

The following signed wrappers already use `message`:
- `SignedAggregateAttestationAndProof` / `SignedAggregateAttestationAndProofElectra`
- `SignedContributionAndProof`
- `SignedBLSToExecutionChange`

## SSZ Compatibility

**Proto field renames do NOT affect SSZ encoding.** SSZ is positional — it uses field order and types, not names. Renaming `block` → `message` keeps the same field number and type, so SSZ wire format is byte-identical.

However, `hack/update-go-ssz.sh` must be re-run after proto changes to regenerate `.ssz.go` files with updated Go field names.

## DB Compatibility

Beacon state and blocks are stored as SSZ-encoded bytes. Since SSZ encoding is unaffected by field renames, existing DB data remains fully compatible.
