# Proto Field Alignment Plan — Step-by-Step PRs

## Principles
- Each PR is independently mergeable and does not break tests
- Order is by risk/impact — smallest scope first
- SSZ wire format is unaffected (positional encoding)
- DB compatibility is maintained (SSZ-encoded storage)
- Each PR must regenerate pb.go and ssz.go after proto changes

## Per-PR Workflow
1. Modify `.proto` field name(s)
2. `hack/update-go-pbs.sh` — regenerate `.pb.go` files
3. `hack/update-go-ssz.sh` — regenerate `.ssz.go` files
4. Rename all Go references (`.FieldName` → `.NewFieldName`)
5. Update `api/server/structs` if affected
6. `bazel run //:gazelle` — update BUILD.bazel
7. `gofmt -w .` + `goimports -w .`
8. `bazel build //...`
9. `bazel test //...`

---

## PR 1: SingleAttestation `committee_id` → `committee_index`

**Scope:** 1 proto field, ~29 Go references
**Risk:** Minimal — Electra-only type, small surface area

Files to modify:
- `proto/prysm/v1alpha1/attestation.proto` — rename field
- All `.go` files referencing `.CommitteeId` → `.CommitteeIndex`
- Note: `CommitteeIndex` type alias already exists in `primitives` — no name collision expected since the field becomes `CommitteeIndex` (matching the cast type)

---

## PR 2: ProposerSlashing `header_1/2` → `signed_header_1/2`

**Scope:** 2 proto fields, ~82 Go references
**Risk:** Low — straightforward rename

Files to modify:
- `proto/prysm/v1alpha1/beacon_core_types.proto` — rename fields
- All `.go` files referencing `.Header_1` → `.SignedHeader_1`, `.Header_2` → `.SignedHeader_2`

---

## PR 3: SyncCommittee `block_root` → `beacon_block_root`

**Scope:** 2 proto fields, selective Go rename
**Risk:** Low-medium — must only rename `BlockRoot` on `SyncCommitteeMessage` and `SyncCommitteeContribution`, not other types

Files to modify:
- `proto/prysm/v1alpha1/sync_committee.proto` — rename fields in both messages
- Selective Go rename: only `.BlockRoot` on these two types → `.BeaconBlockRoot`
- **Caution:** Other types (e.g., `BlobSidecar`) also have `block_root` which is correct per spec — do NOT rename those

---

## PR 4: Validator/Deposit `public_key` → `pubkey`

**Scope:** 2 proto fields, ~265 Go references
**Risk:** Medium — wide usage but mechanical rename

Files to modify:
- `proto/prysm/v1alpha1/beacon_state.proto` — `Validator.public_key` → `pubkey`
- `proto/prysm/v1alpha1/beacon_core_types.proto` — `Deposit.Data.public_key` → `pubkey`
- All `.go` files: `.PublicKey` → `.Pubkey` on Validator and Deposit.Data
- `spec_name` annotation already exists — can be removed after rename

---

## PR 5: AttestationData `committee_index` → `index`

**Scope:** 1 proto field, ~380 Go references
**Risk:** Medium — widely used type, but no ambiguity since `AttestationData` context is always clear

Files to modify:
- `proto/prysm/v1alpha1/attestation.proto` — rename field
- All `.go` files: `.CommitteeIndex` → `.Index` on AttestationData
- **Note:** Must not rename `CommitteeIndex` on other types (e.g., `SingleAttestation` which we already fixed in PR 1)

---

## PR 6: SignedVoluntaryExit `exit` → `message`

**Scope:** 1 proto field, moderate Go references
**Risk:** Medium — `exit` is a Go keyword-adjacent name, `message` is clearer

Files to modify:
- `proto/prysm/v1alpha1/beacon_core_types.proto` — rename field
- All `.go` files: `.Exit` → `.Message` on SignedVoluntaryExit

---

## PR 7: SignedBeaconBlockHeader `header` → `message`

**Scope:** 1 proto field
**Risk:** Medium — used in ProposerSlashing and other contexts

Files to modify:
- `proto/prysm/v1alpha1/beacon_core_types.proto` — rename field
- All `.go` files: `.Header` → `.Message` on SignedBeaconBlockHeader

---

## PR 8: SignedBeaconBlock `block` → `message` (all fork variants)

**Scope:** 8 proto fields (one per fork), ~3,100 Go references
**Risk:** High — largest rename, touches nearly every package

Files to modify:
- `proto/prysm/v1alpha1/beacon_block.proto` — rename `block` → `message` in all 8 `SignedBeaconBlock*` messages
- All `.go` files: `.Block` → `.Message` on all SignedBeaconBlock variants
- **Warning:** `.Block` is extremely common — must use type-aware renaming, not blind find-replace
- Consider splitting into sub-PRs per fork if too large

---

## Summary

| PR | Change | Refs | Risk |
|---|---|---|---|
| 1 | `SingleAttestation.committee_id` → `committee_index` | 29 | ✅ Minimal |
| 2 | `ProposerSlashing.header_1/2` → `signed_header_1/2` | 82 | ✅ Low |
| 3 | `SyncCommittee*.block_root` → `beacon_block_root` | ~50 | ✅ Low |
| 4 | `Validator/Deposit.public_key` → `pubkey` | 265 | ⚠️ Medium |
| 5 | `AttestationData.committee_index` → `index` | 380 | ⚠️ Medium |
| 6 | `SignedVoluntaryExit.exit` → `message` | ~100 | ⚠️ Medium |
| 7 | `SignedBeaconBlockHeader.header` → `message` | ~50 | ⚠️ Medium |
| 8 | `SignedBeaconBlock*.block` → `message` (8 forks) | 3,100 | 🔴 High |

**Total: 22 field renames across 8 PRs**
