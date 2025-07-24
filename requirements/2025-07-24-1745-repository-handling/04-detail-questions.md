# Expert Technical Questions - Repository Service

## Q1: Should the repository service extend the existing `backend/ent/schema/repository.go` to include cloud-specific fields like `borgbase_id` and `ssh_key_fingerprint`?
**Default if unknown:** Yes (maintaining single repository entity with cloud metadata fields)

## Q2: Should cloud repository passwords be stored locally in the existing `password` field or completely omitted for security?
**Default if unknown:** Omitted (cloud repositories handle encryption, storing passwords locally creates security risk)

## Q3: Should the repository service implement RPC server handlers (UnimplementedRepositoryServiceHandler) for external cloud requests?
**Default if unknown:** No (repository service is primarily a client to external BorgBase cloud service)

## Q4: Should cloud repository operations integrate with the existing `RepoState` and repository locking mechanisms in `backend/app/repository.go`?
**Default if unknown:** Yes (maintains consistent repository management experience regardless of type)

## Q5: Should the repository service maintain compatibility with existing backup profiles that reference local repositories by supporting both local and cloud repositories in the same backup profile?
**Default if unknown:** Yes (users should be able to backup to both local and cloud repositories from same profile)