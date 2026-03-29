# S02: Domain Model + Status Enums

**Goal:** Typed Go domain model in `internal/auto/` wrapping SQLC-generated DB structs with Status/Phase enums and bidirectional conversion functions.
**Demo:** After this: TBD

## Tasks
- [x] **T01: Created internal/auto/ package with Status/Phase enums and Milestone, Slice, Task domain structs with bidirectional DB conversion functions** — 
  - Files: internal/auto/status.go, internal/auto/milestone.go, internal/auto/slice.go, internal/auto/task.go
  - Verify: go build ./internal/auto/... && go vet ./internal/auto/...
- [x] **T02: Added 16 parallel tests for enum validity, DB value matching, and round-trip conversion in internal/auto/** — 
  - Files: internal/auto/status_test.go, internal/auto/model_test.go
  - Verify: go test ./internal/auto/ -v -count=1 && go test ./internal/db/ -run TestAuto -v -count=1
