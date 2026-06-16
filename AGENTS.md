# AGENTS.md

Guidance for AI agents working in this repository.

`github.com/jkbrsn/jsonrpc` is a JSON-RPC 2.0 implementation in Go (1.26+).
It uses `bytedance/sonic` for JSON and exposes three core types — `Request`,
`Response`, `Error` — plus batch helpers. Public API lives in the repo root to
mirror the import path.

## Commands

```bash
go test ./...                          # all tests
go test ./... -run TestResponseParse   # single test
go test ./... -cover                   # coverage
```

## Architecture

Core types: `Request` (request.go), `Response` (response.go), `Error` (error.go).
Batch encode/decode helpers live in `batch.go` (`DecodeRequestOrBatch`,
`EncodeBatchRequest`, `NewBatchRequest`, …).

### Response: parse once, access many

`Response` defers parsing for performance and thread-safety:
- Raw fields (`rawID`, `rawError`, `result`) held as `json.RawMessage`.
- Lazy access via `IDRaw()`, `UnmarshalError()`, `UnmarshalResult(dst)`, guarded
  by `sync.Once`; an `ast.Node` is cached (under RWMutex) for field access.
- Constructors: `NewResponse`, `NewResponseFromRaw`.

### JSON handling

All marshal/unmarshal goes through `getSonicAPI()` (performance.go), whose
behavior is selectable via `SetPerformanceProfile` (`ProfileDefault` …
`ProfileAggressive`). Raw fields use `encoding/json.RawMessage` so callers can
decode with either sonic or stdlib. Custom `UnmarshalJSON` validates fields;
`MarshalJSON` and the streaming `WriteTo` path emit a JSON-RPC 2.0 structure.

## Spec deviations

- Fractional IDs allowed (spec says "should not").
- Flexible error unmarshaling for custom error handling.
- Zero error codes are valid per spec, but `IsEmpty()` treats them as empty
  when the message is also empty.
