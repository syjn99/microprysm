# Proto Field Name vs Beacon API Spec — Investigation

## Background

Prysm's protobuf definitions use field names that diverge from the [Ethereum Beacon API spec](https://ethereum.github.io/beacon-APIs/) and [consensus-specs](https://github.com/ethereum/consensus-specs). This causes JSON serialization mismatches when using codegen tools like `prysm-jsongen` that derive JSON field names from proto field names.

Discovered via `prysm-jsongen` oracle comparison tests ([CI run](https://github.com/syjn99/prysm-jsongen/actions/runs/22812318921/job/66171261893)).

## Approach: `spec_name` Annotations

Instead of renaming proto fields (which would require thousands of Go code changes), we leverage the existing `ethereum.eth.ext.spec_name` proto extension to annotate the spec-compliant JSON field name. `prysm-jsongen` reads this annotation and uses it as the JSON key.

This approach:
- **Zero Go code changes** — proto field names (and thus Go struct fields) stay the same
- **SSZ unaffected** — `spec_name` is already used by SSZ codegen for hash tree root field names
- **DB compatible** — SSZ encoding is positional, not name-based
- **Enables prysm-jsongen** — codegen reads `spec_name` for JSON field naming

## Discrepancies Found: 22 fields across 6 groups

### Group 1: Signed wrapper inner field → `message` (13 fields)

The Beacon API spec uses `message` for the inner object of `Signed*` wrapper types.

| Proto Message | Proto Field | Spec Field (spec_name) |
|---|---|---|
| `SignedBeaconBlock` (Phase0–Gloas, 9 variants) | `block` | `message` |
| `SignedBeaconBlockHeader` | `header` | `message` |
| `SignedVoluntaryExit` | `exit` | `message` |

**Note:** `SignedBlindedBeaconBlock{Deneb,Electra,Fulu}` already use `message`. `SignedAggregate*` and `SignedContributionAndProof` also already correct.

### Group 1b: SignedBlockContents → `signed_block` (3 fields)

| Proto Message | Proto Field | Spec Field (spec_name) |
|---|---|---|
| `SignedBeaconBlockContentsDeneb` | `block` | `signed_block` |
| `SignedBeaconBlockContentsElectra` | `block` | `signed_block` |
| `SignedBeaconBlockContentsFulu` | `block` | `signed_block` |

### Group 2: `AttestationData.committee_index` → `index` (1 field)

| Proto Message | Proto Field | Spec Field |
|---|---|---|
| `AttestationData` | `committee_index` | `index` |

### Group 3: `block_root` → `beacon_block_root` (2 fields)

| Proto Message | Proto Field | Spec Field |
|---|---|---|
| `SyncCommitteeMessage` | `block_root` | `beacon_block_root` |
| `SyncCommitteeContribution` | `block_root` | `beacon_block_root` |

### Group 4: `SingleAttestation.committee_id` → `committee_index` (1 field)

| Proto Message | Proto Field | Spec Field |
|---|---|---|
| `SingleAttestation` | `committee_id` | `committee_index` |

### Group 5: `ProposerSlashing.header_1/2` → `signed_header_1/2` (2 fields)

| Proto Message | Proto Field | Spec Field |
|---|---|---|
| `ProposerSlashing` | `header_1` | `signed_header_1` |
| `ProposerSlashing` | `header_2` | `signed_header_2` |

### Group 6: `public_key` → `pubkey` (already annotated)

Already has `spec_name = "pubkey"` on all instances. No changes needed.

## Files Modified

- `proto/prysm/v1alpha1/beacon_block.proto` — 12 fields
- `proto/prysm/v1alpha1/beacon_core_types.proto` — 4 fields
- `proto/prysm/v1alpha1/attestation.proto` — 2 fields
- `proto/prysm/v1alpha1/sync_committee.proto` — 2 fields
- `proto/prysm/v1alpha1/gloas.proto` — 1 field

Total: **21 new `spec_name` annotations** (+ 10 pre-existing `pubkey` annotations)
