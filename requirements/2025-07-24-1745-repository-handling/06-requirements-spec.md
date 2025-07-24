# Requirements Specification - Repository Handling in ArcoCloud

## Problem Statement

Arco currently handles local Borg repositories through the existing `RepositoryClient` in `backend/app/repository.go`, but lacks integration with ArcoCloud-managed repositories. The `proto/api/v1/repository.proto` file defines a comprehensive RepositoryService for cloud repository management but is not yet incorporated into the application architecture.

## Solution Overview

Implement a new cloud repository service following the established Service/ServiceInternal pattern that:
- Manages ArcoCloud repositories via Connect RPC
- Extends the existing repository schema to support cloud-specific metadata
- Integrates with existing repository state management and backup scheduling
- Maintains separation between local and cloud repository operations
- Supports hybrid backup profiles with both local and cloud repositories

## Functional Requirements

### Core Repository Operations
1. **Add Repository**: Create new ArcoCloud repositories with name, password, SSH key, and location
2. **Delete Repository**: Permanently remove cloud repositories and all backup data
3. **List Repositories**: Retrieve all cloud repositories owned by authenticated user
4. **Get Repository**: Fetch detailed information about specific cloud repositories
5. **Replace SSH Key**: Update SSH access keys for existing cloud repositories

### Authentication & Security
1. **JWT Authentication**: All cloud repository operations require authenticated user session
2. **SSH Key Management**: Store SSH key fingerprints locally, validate SSH key formats
3. **Password Handling**: Store repository passwords in existing `password` field
4. **Rate Limiting**: Respect ArcoCloud rate limits for repository creation operations

### Integration Requirements
1. **State Management**: Cloud repositories integrate with existing `RepoState` and locking mechanisms
2. **Backup Profile Compatibility**: Support mixed local/cloud repositories in same backup profile  
3. **UI Consistency**: Cloud repositories appear seamlessly alongside local repositories in frontend
4. **Event Synchronization**: Use existing event patterns for repository state changes

### Repository Type Differentiation
1. **Type Detection**: Distinguish cloud repositories by ArcoCloud URL patterns
2. **Operation Routing**: Route repository operations to appropriate handler (local vs cloud)
3. **Metadata Management**: Store cloud-specific fields (arco_cloud_id, ssh_key_fingerprint) in extended schema

## Technical Requirements

### Backend Implementation

#### 1. Repository Service Structure (`backend/app/repository/repository_service.go`)
```go
// Service contains business logic exposed to frontend
type Service struct {
    log       *zap.SugaredLogger
    db        *ent.Client
    state     *state.State
    rpcClient arcov1connect.RepositoryServiceClient
}

// ServiceInternal wraps Service and adds RPC handlers
type ServiceInternal struct {
    *Service
    arcov1connect.UnimplementedRepositoryServiceHandler
}
```

#### 2. Service Registration Pattern (`backend/app/app.go`)
```go
// Add to App struct
repositoryService *repository.ServiceInternal

// Constructor initialization
repositoryService: repository.NewService(log, state),

// Database and RPC client injection in Startup()
repositoryService.Init(db, repositoryRPCClient)

// Expose frontend-facing service
func (a *App) RepositoryService() *repository.Service {
    return a.repositoryService.Service
}
```

#### 3. Database Schema Extensions (`backend/ent/schema/repository.go`)
```go
// Add cloud-specific fields to existing repository schema
field.String("arco_cloud_id").
    StructTag(`json:"arcoCloudId"`).
    Optional().
    Nillable(),
field.String("ssh_key_fingerprint").
    StructTag(`json:"sshKeyFingerprint"`).
    Optional().
    Nillable(),
```

#### 4. Service Method Implementation Patterns
```go
// Frontend-exposed business logic methods
func (s *Service) AddCloudRepository(ctx context.Context, name, password, sshKey string, location arcov1.RepositoryLocation) (*ent.Repository, error) {
    s.mustHaveDB()
    
    // Call cloud service
    req := connect.NewRequest(&arcov1.AddRepositoryRequest{...})
    resp, err := s.rpcClient.AddRepository(ctx, req)
    if err != nil {
        return nil, s.handleRPCError("add repository", err)
    }
    
    // Create local entity with cloud metadata
    return s.syncCloudRepository(ctx, resp.Msg.Repository)
}

// RPC server handler implementation
func (si *ServiceInternal) AddRepository(ctx context.Context, req *connect.Request[arcov1.AddRepositoryRequest]) (*connect.Response[arcov1.AddRepositoryResponse], error) {
    // Implementation for incoming cloud requests
}
```

#### 5. Repository Type Management
```go
func (s *Service) isCloudRepository(repo *ent.Repository) bool {
    return repo.ArcoCloudID != nil && *repo.ArcoCloudID != ""
}

func (s *Service) DeleteRepository(ctx context.Context, repoID int) error {
    repo, err := s.db.Repository.Get(ctx, repoID)
    if err != nil {
        return err
    }
    
    if s.isCloudRepository(repo) {
        // Handle cloud repository deletion
        return s.deleteCloudRepository(ctx, repo)
    }
    
    // Delegate to existing RepositoryClient for local repositories
    return s.repositoryClient.Delete(repoID)
}
```

### Frontend Implementation

#### 1. Generated Bindings
- Auto-generate TypeScript bindings via `task proto:generate`
- Bindings location: `frontend/bindings/github.com/loomi-labs/arco/backend/app/repository/`

#### 2. Service Composable Pattern (`frontend/src/common/repository.ts`)
```typescript
export function useCloudRepositoryService() {
  const isLoading = ref(false);
  const errorMessage = ref<string | undefined>(undefined);

  async function createCloudRepository(data: CreateCloudRepositoryRequest) {
    try {
      isLoading.value = true;
      const result = await RepositoryService.AddCloudRepository(data);
      return result;
    } catch (error) {
      await showAndLogError("Failed to create cloud repository", error);
      throw error;
    } finally {
      isLoading.value = false;
    }
  }

  return {
    isLoading: computed(() => isLoading.value),
    errorMessage: computed(() => errorMessage.value),
    createCloudRepository
  };
}
```

#### 3. Component Integration
- Extend existing repository management components
- Use consistent loading/error state patterns
- Implement repository type indicators in UI
- Follow existing modal and form validation patterns

## Implementation Phases

### Phase 1: Backend Service Foundation
1. Create `backend/app/repository/repository_service.go` with Service/ServiceInternal structure
2. Implement basic RPC client methods (AddRepository, DeleteRepository, ListRepositories, GetRepository, ReplaceSSHKey)
3. Add service registration to `backend/app/app.go`
4. Extend `backend/ent/schema/repository.go` with cloud-specific fields
5. Generate database migrations for schema changes

### Phase 2: Integration with Existing Systems
1. Implement repository type detection and operation routing
2. Integrate cloud repositories with existing `RepoState` and locking mechanisms
3. Ensure backup profile compatibility for mixed repository types
4. Add comprehensive error handling and Connect RPC error mapping

### Phase 3: Frontend Integration
1. Generate TypeScript bindings via `task proto:generate`
2. Create cloud repository service composables
3. Extend existing repository management UI components
4. Implement cloud repository creation/management flows
5. Add repository type indicators and appropriate UI patterns

### Phase 4: Testing and Validation
1. Test all cloud repository operations (CRUD, SSH key management)
2. Verify proper integration with backup scheduling and monitoring
3. Validate mixed local/cloud repository scenarios in backup profiles
4. Ensure consistent state management and event emission
5. Test authentication requirements and error handling

## Acceptance Criteria

### Functional Acceptance
- [ ] Users can create cloud repositories through desktop UI with name, password, SSH key, and location
- [ ] Users can view all repositories (local and cloud) in unified repository list
- [ ] Users can delete cloud repositories with proper confirmation and data loss warnings
- [ ] Users can update SSH keys for cloud repositories
- [ ] Users can create backup profiles using both local and cloud repositories
- [ ] Cloud repositories integrate seamlessly with backup scheduling and monitoring

### Technical Acceptance
- [ ] Repository service follows established Service/ServiceInternal pattern
- [ ] All cloud repository operations require authentication
- [ ] Repository schema properly stores cloud-specific metadata
- [ ] Repository type detection correctly routes operations
- [ ] State management and locking work consistently for cloud repositories
- [ ] Frontend bindings are properly generated and functional
- [ ] Error handling provides appropriate user feedback
- [ ] Event emission maintains UI consistency across repository types

### Security Acceptance
- [ ] SSH keys are validated and only fingerprints stored locally
- [ ] Repository passwords are securely handled and stored
- [ ] Authentication tokens are properly used for all cloud operations
- [ ] Rate limiting is respected to prevent ArcoCloud abuse

## Assumptions

1. **ArcoCloud Service Availability**: ArcoCloud backend service implementing the repository.proto API is available and functional
2. **Authentication Integration**: Existing JWT authentication system works with ArcoCloud repository service
3. **Network Connectivity**: Cloud repository operations require internet connectivity
4. **Backwards Compatibility**: Existing local repository functionality remains unchanged
5. **Schema Migration**: Database migrations can be safely applied to existing repository data
6. **UI Framework**: Frontend uses existing Vue 3 + TypeScript patterns and can be extended for cloud repositories

## Dependencies

- **Generated Code**: `backend/api/v1/arcov1connect/repository.connect.go` (already exists)
- **Proto Compilation**: `task proto:generate` must be run after any proto changes
- **Database Migration**: `task db:migrate:new` and `task db:migrate` for schema changes
- **Frontend Bindings**: `task common:generate:bindings` for TypeScript binding generation
- **Authentication**: Existing JWT authentication system for cloud service access