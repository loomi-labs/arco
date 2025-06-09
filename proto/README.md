# Protocol Buffers

This directory contains the Protocol Buffer definitions for the Arco Cloud API.

## Structure

```text
proto/
├── buf.yaml               # Buf module configuration
├── buf.gen.yaml           # Code generation configuration
├── buf.lock               # Dependency lock file (auto-generated)
└── api/v1/                # Proto files (matches package structure)
    ├── auth.proto         # Authentication service definitions
    ├── payment.proto      # Payment service definitions
    ├── subscription.proto # Subscription service definitions
    └── user.proto         # User service definitions
```

## Generated Files

Generated Go code is output to:
- `backend/api/v1/*.pb.go` - Protocol buffer messages
- `backend/api/v1/arcov1connect/*.connect.go` - Connect-RPC service handlers

## Commands

All proto-related tasks are defined in `Taskfile.proto.yml`:

- `task proto:generate` - Generate Go code from proto definitions
- `task proto:lint` - Lint proto files
- `task proto:format` - Format proto files
- `task proto:breaking` - Check for breaking changes (requires committed proto files)
- `task proto:clean` - Remove generated files

## Adding New Services

1. Create a new `.proto` file to `proto/api/v1/`
2. Ensure the package is `api.v1`
3. Run `task proto:generate` to generate the Go code
4. Implement the service interface in `backend/internal/<service>/`
5. Add the new service to the server in `backend/cmd/server/main.go`:
