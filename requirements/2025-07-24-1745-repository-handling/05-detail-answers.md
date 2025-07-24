# Expert Technical Answers

## Q1: Should the repository service extend the existing `backend/ent/schema/repository.go` to include cloud-specific fields like `borgbase_id` and `ssh_key_fingerprint`?
**Answer:** Yes (but borgbase entries should be renamed to arco-cloud)

## Q2: Should cloud repository passwords be stored locally in the existing `password` field or completely omitted for security?
**Answer:** in the existing password field

## Q3: Should the repository service implement RPC server handlers (UnimplementedRepositoryServiceHandler) for external cloud requests?
**Answer:** The one that we will create? Yes; The existing RepositoryClient in @backend/app/repository.go -> No... lets separate those in the beginning

## Q4: Should cloud repository operations integrate with the existing `RepoState` and repository locking mechanisms in `backend/app/repository.go`?
**Answer:** yes

## Q5: Should the repository service maintain compatibility with existing backup profiles that reference local repositories by supporting both local and cloud repositories in the same backup profile?
**Answer:** yes