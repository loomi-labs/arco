# Context Findings - Repository Service Implementation

## Architecture Patterns Identified

### Service Structure Pattern
- **Service/ServiceInternal Pattern**: All services (auth, plan, subscription) follow this pattern
  - `Service` struct contains business logic exposed to frontend
  - `ServiceInternal` wraps Service and adds backend-only methods
  - Two-phase initialization: constructor + Init() method
  
### Service Registration in app.go
```go
// Phase 1: Constructor during App creation
repositoryService: repository.NewService(log, state),

// Phase 2: Database and RPC client injection during Startup
repositoryService.Init(db, repositoryRPCClient)
```

### Frontend Integration Patterns
- **Generated Bindings**: Services imported from `frontend/bindings/github.com/loomi-labs/arco/backend/app/`
- **Composable Pattern**: Service logic wrapped in composables like `useAuth()` for reusability
- **Loading/Error States**: Consistent reactive state management with `isLoading` and `errorMessage`
- **Event-Driven Sync**: Use global events for state synchronization between frontend/backend
- **No Credential Storage**: Frontend never stores sensitive data, passes directly to backend

### Database Integration Patterns
- **Hybrid Management**: Repository service must handle both local and cloud repositories
- **Safety Guards**: All services use `mustHaveDB()` to prevent nil database access
- **Entity Bridging**: Cloud repositories sync with local `ent.Repository` entities

### RPC Client Patterns
- **Authenticated Clients**: Use JWT interceptor for cloud service authentication
- **Connect RPC Framework**: Use `connect.NewRequest()` wrapper pattern
- **Error Mapping**: Map Connect error codes to domain-specific status types
- **Context Propagation**: Always pass context through all service calls

## Specific Files to Create/Modify

### Backend Files
1. **NEW**: `backend/app/repository/repository_service.go` - Main service implementation
2. **MODIFY**: `backend/app/app.go` - Add repository service registration
3. **EXTEND**: `backend/ent/schema/repository.go` - May need cloud repository fields

### Frontend Files
1. **AUTO-GENERATED**: `frontend/bindings/github.com/loomi-labs/arco/backend/app/repository/`
2. **MODIFY**: Existing repository management components to use new service
3. **EXTEND**: `frontend/src/common/repository.ts` - Add cloud repository utilities

## Key Integration Points

### Sensitive Data Handling
- **SSH Keys**: Only send to cloud, store fingerprint locally
- **Passwords**: Cloud repositories should NOT store passwords locally
- **Authentication**: Repository operations require authenticated RPC client

### Repository Type Management
- **Local Repositories**: Existing pattern with local Borg installations
- **Cloud Repositories**: BorgBase URLs (`@repo.borgbase.com:`)
- **Hybrid Operations**: Service must handle both types appropriately

### State Management
- **Repository States**: Cloud repositories integrate with existing `RepoState` patterns
- **Event Emission**: Use same events for UI consistency regardless of repository type
- **Background Operations**: Support streaming patterns for long-running operations

## Similar Features Analyzed
- **Auth Service**: Magic link authentication with secure credential handling
- **Plan Service**: Simple RPC forwarding pattern for cloud data
- **Subscription Service**: Complex state management with background monitoring
- **Repository Client**: Existing local repository management patterns

## Technical Constraints
- **BorgBase Integration**: Repository service specifically designed for BorgBase cloud storage
- **Geographic Locations**: Support EU/US regions for data sovereignty
- **Rate Limiting**: Repository creation has rate limits to prevent BorgBase abuse
- **Authentication Required**: All repository operations require authenticated user
- **Connect RPC Only**: No gRPC, uses Connect framework for cloud communication

## Implementation Hints
1. Follow two-phase service initialization pattern exactly
2. Use existing `ent.Repository` schema as base, extend if needed for cloud fields
3. Implement repository type detection for hybrid local/cloud management
4. Follow Connect RPC error mapping patterns from auth service
5. Use event-driven state synchronization for real-time UI updates
6. Generate TypeScript bindings using existing `task proto:generate` command