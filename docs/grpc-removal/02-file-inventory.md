# gRPC File Inventory — Files to Modify or Delete

## Files to DELETE entirely

### validator/client/grpc-api/ (Phase 3.3)
- grpc_validator_client.go
- grpc_validator_client_test.go
- grpc_node_client.go
- grpc_node_client_test.go
- grpc_beacon_chain_client.go
- grpc_prysm_beacon_chain_client.go
- grpc_prysm_beacon_chain_client_test.go
- grpc_client_manager.go
- grpc_client_manager_test.go

### beacon-chain/rpc/prysm/v1alpha1/ (Phase 4.2-4.5)
- validator/server.go, server_test.go
- validator/duties.go, duties_v2.go
- validator/status.go, status_test.go
- validator/proposer.go, proposer_test.go
- validator/blocks.go, blocks_test.go
- validator/attester.go, attester_test.go
- validator/aggregator.go
- validator/sync_committee.go, sync_committee_test.go
- validator/exit.go
- validator/proposer_deposits.go
- validator/proposer_payload_envelope.go
- validator/proposer_empty_block.go
- beacon/blocks.go, attestations.go, committees.go, assignments.go, validators.go, slashings.go
- node/server.go, server_test.go
- debug/server.go, p2p.go, state.go, block.go

### api/grpc/ (Phase 5.1)
- grpc_connection_provider.go
- grpc_connection_provider_test.go
- grpcutils.go
- grpcutils_test.go
- mock_grpc_provider.go

### testing/mock/ (Phase 5.2)
- beacon_service_mock.go
- beacon_validator_client_mock.go
- beacon_validator_server_mock.go
- beacon_altair_validator_client_mock.go
- beacon_altair_validator_server_mock.go
- node_service_mock.go

### proto/eth/v1/ (Phase 6.4)
- attestation.proto, attestation.pb.go
- beacon_block.proto, beacon_block.pb.go
- beacon_chain.proto, beacon_chain.pb.go
- events.proto, events.pb.go
- node.proto, node.pb.go
- validator.proto, validator.pb.go
- data_columns.proto
- gateway.ssz.go

### tools/ (Phase 2.1)
- ~~tools/forkchecker/forkchecker.go~~ ✅ DELETED (PR #5)

## Files to MODIFY

### Phase 2
- cmd/prysmctl/p2p/client.go — rewrite to REST
- cmd/validator/accounts/exit.go — ✅ rewritten to REST (PR #9)

### Phase 3
- config/features/config.go — flip/remove EnableBeaconRESTApi
- config/features/flags.go — remove flag definition
- validator/client/validator-client-factory/validator_client_factory.go — always REST
- validator/client/node-client-factory/node_client_factory.go — always REST
- validator/client/beacon-chain-client-factory/beacon_chain_client_factory.go — always REST
- validator/helpers/node_connection.go — remove gRPC options
- validator/client/service.go — remove gRPC dial options, ConstructDialOptions()
- validator/rpc/beacon.go — remove gRPC middleware
- validator/rpc/intercepter.go — simplify or delete
- validator/accounts/cli_manager.go — remove gRPC config
- validator/accounts/cli_options.go — remove gRPC config

### Phase 4
- beacon-chain/rpc/service.go — remove gRPC server entirely

### Phase 5
- beacon-chain/rpc/core/errors.go — replace gRPC status codes
- beacon-chain/rpc/core/duties.go — replace gRPC status codes
- beacon-chain/rpc/eth/helpers/error_handling.go — replace gRPC status conversion
- proto/prysm/v1alpha1/validator.proto — remove service block
- proto/prysm/v1alpha1/beacon_chain.proto — remove service block
- proto/prysm/v1alpha1/node.proto — remove service block
- proto/prysm/v1alpha1/health.proto — remove service block
- proto/prysm/v1alpha1/debug.proto — remove service block

### Phase 6
- proto/migration/v1alpha1_to_v1.go — remove ethv1 conversions
- beacon-chain/blockchain/receive_block.go — remove ethv1
- beacon-chain/blockchain/head.go — remove ethv1
- beacon-chain/rpc/eth/events/events.go — remove ethv1
- beacon-chain/rpc/eth/helpers/ — remove ethv1
- beacon-chain/rpc/eth/node/ — remove ethv1
- beacon-chain/p2p/ — remove ethv1
- testing/util/attestation.go, block.go — remove ethv1
- api/server/structs/conversions.go — remove ethv1
- consensus-types/mock/ — remove ethv1

### Phase 7
- go.mod — remove gRPC dependencies
- go.sum — regenerated
