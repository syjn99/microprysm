# Risk Analysis & Verification Strategy

## Key Risks

### 1. REST API Parity Gaps
**Risk:** Some gRPC endpoints may not have REST equivalents.
**Mitigation:** Before each Phase 4 PR, audit REST endpoints in `beacon-chain/rpc/endpoints.go` against the gRPC service being removed. Known gaps:
- `StreamBeaconLogs` (Health service) — may need SSE equivalent
- `SetLoggingLevel` (Debug service) — may need REST endpoint
- Peer scoring info — `PeersCheck` evaluator needed custom REST endpoint (already added on this branch)

### 2. proto/prysm/v1alpha1 Message Types
**Risk:** Confusion between removing gRPC services vs. message types.
**Clarification:** We KEEP all protobuf message types. Only `service` blocks and their generated `_grpc.pb.go` stubs are removed. The 1,110+ files importing v1alpha1 messages are unaffected.

### 3. Validator Client Migration
**Risk:** REST implementation may have subtle behavioral differences from gRPC.
**Mitigation:**
- PR 3.1 (flip default) should be tested extensively with E2E
- Monitor for: timeout differences, error handling differences, SSZ vs JSON encoding issues
- The existing fallback pattern (PR 3.2) provides a safety net during transition

### 4. External Consumers
**Risk:** External tools/scripts may use gRPC API directly.
**Mitigation:** This is a microprysm fork — external consumer risk is minimal. Document breaking change in release notes.

### 5. Web UI
**Risk:** Per James' comment, web UI may depend on gRPC.
**Mitigation:** Audit `validator/rpc/` handlers — these serve the web UI via HTTP REST, not gRPC. The web UI calls REST endpoints, not gRPC directly. Verify before Phase 3.

## Verification Strategy

### Per-PR Checklist
1. `go build ./...` — compilation passes
2. `go test ./... -tags=develop` — unit tests pass
3. No import cycles introduced
4. Gazelle/BUILD.bazel files updated
5. E2E tests pass (for Phase 3+ PRs)

### Phase Gates
- **After Phase 1:** All existing tests still pass (dead code only)
- **After Phase 2:** CLI tools work with REST endpoints
- **After Phase 3:** Validator client operates fully on REST — run full E2E
- **After Phase 4:** Beacon node runs without gRPC server — run full E2E
- **After Phase 5:** No gRPC imports remain — `grep -r "google.golang.org/grpc" --include="*.go"` returns empty
- **After Phase 7:** `go mod tidy` removes all gRPC transitive deps

## Dependency Graph
```
Phase 1 ──┐
           ├──→ Phase 3 ──→ Phase 4 ──→ Phase 5 ──→ Phase 7
Phase 2 ──┘                    │
                               └──→ Phase 6 ──→ Phase 7
```
