#!/bin/bash
. "$(dirname "$0")"/common.sh

# Script to regenerate JSON marshal/unmarshal code for proto types using prysm-jsongen.
# Install jsongen: go install github.com/syjn99/prysm-jsongen/cmd/jsongen@latest

JSONGEN="${JSONGEN:-$(command -v jsongen)}"
if [ -z "$JSONGEN" ] || [ ! -x "$JSONGEN" ]; then
    color "31" "jsongen not found. Install with: go install github.com/syjn99/prysm-jsongen/cmd/jsongen@latest"
    exit 1
fi

# Proto package and types to generate JSON marshaling for.
PACKAGE="github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
TYPES=$(
    IFS=,
    echo "VoluntaryExit,Checkpoint,Fork,AttestationData,BeaconBlockHeader,SignedBeaconBlockHeader,SignedVoluntaryExit,ProposerSlashing,Eth1Data,Validator,IndexedAttestation,AttesterSlashing,SyncCommittee,BLSToExecutionChange,SignedBLSToExecutionChange,HistoricalSummary,BeaconBlock,BeaconBlockBody,Deposit,Deposit_Data,Attestation,SyncAggregate,BeaconState,PendingAttestation,SingleAttestation,AttestationElectra,IndexedAttestationElectra,AttesterSlashingElectra,PendingDeposit,PendingPartialWithdrawal,PendingConsolidation,SyncCommitteeContribution,ContributionAndProof,SignedContributionAndProof,ValidatorRegistrationV1,SignedValidatorRegistrationV1,PayloadAttestation,PayloadAttestationData,ExecutionPayloadBid,SignedExecutionPayloadBid,Builder,BuilderPendingPayment,BuilderPendingWithdrawal,SyncCommitteeMessage,BeaconBlockAltair,BeaconBlockBodyAltair,SignedBeaconBlockAltair,SignedBeaconBlock"
)
OUTPUT_DIR="proto/prysm/v1alpha1"

color "34" "Generating JSON marshal/unmarshal code..."
"$JSONGEN" \
    --package "$PACKAGE" \
    --types "$TYPES" \
    --output "$OUTPUT_DIR/"

if [ $? -eq 0 ]; then
    color "32" "JSON generation complete: $OUTPUT_DIR/eth_json.go"
    goimports -w "$OUTPUT_DIR/eth_json.go" 2>/dev/null || true
    gofmt -w "$OUTPUT_DIR/eth_json.go"
else
    color "31" "JSON generation failed"
    exit 1
fi
