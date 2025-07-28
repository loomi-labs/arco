package app

import (
	"fmt"
	"github.com/loomi-labs/arco/backend/ent/archive"
	"github.com/loomi-labs/arco/backend/ent/repository"
	"github.com/loomi-labs/arco/backend/ent/schema"
	"github.com/negrel/assert"
	"regexp"
	"strings"
)

// TODO: Remove this cross-service dependency - validation should not depend on repository client

func (v *ValidationClient) backupClient() *BackupClient {
	return (*BackupClient)(v)
}

// ArchiveName validates the name of an archive.
// The rules are not enforced by the database because we import them from borg repositories which have different rules.
func (v *ValidationClient) ArchiveName(archiveId int, prefix, name string) (string, error) {
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < 3 {
		return "Name must be at least 3 characters long", nil
	}
	if len(name) > 50 {
		return "Name can not be longer than 50 characters", nil
	}
	pattern := `^[a-zA-Z0-9-_]+$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return "", err
	}
	if !matched {
		return "Name can only contain letters, numbers, hyphens, and underscores", nil
	}

	// TODO: Remove cross-service dependency - validation should not directly access repository service
	// For now, this will cause a compilation error that needs to be fixed properly
	arch, err := (*App)(v).RepositoryService().GetArchive(v.ctx, archiveId)
	if err != nil {
		return "", err
	}
	assert.NotNil(arch.Edges.Repository, "archive must have a repository")

	// Check if the new name starts with the backup profile prefix
	if arch.Edges.BackupProfile != nil {
		if !strings.HasPrefix(prefix, arch.Edges.BackupProfile.Prefix) {
			return "The new name must start with the backup profile prefix", nil
		}
	} else {
		if prefix != "" {
			err = fmt.Errorf("the archive can not have a prefix if it is not connected to a backup profile")
			assert.Error(err)
			return "", err
		}

		// If it is not connected to a backup profile,
		// it can not start with any prefix used by another backup profile of the repository
		backupProfiles, err := arch.Edges.Repository.QueryBackupProfiles().All(v.ctx)
		if err != nil {
			return "", err
		}
		for _, bp := range backupProfiles {
			prefixWithoutTrailingDash := strings.TrimSuffix(bp.Prefix, "-")
			if strings.HasPrefix(name, prefixWithoutTrailingDash) {
				return "The new name must not start with the prefix of another backup profile", nil
			}
		}
	}

	fullName := prefix + name
	exist, err := v.db.Archive.
		Query().
		Where(archive.Name(fullName)).
		Where(archive.HasRepositoryWith(repository.ID(arch.Edges.Repository.ID))).
		Exist(v.ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Archive name must be unique", nil
	}

	return "", nil
}

// RepoName validates the name of a repository.
// The rules are enforced by the database.
func (v *ValidationClient) RepoName(name string) (string, error) {
	if name == "" {
		return "Name is required", nil
	}
	if len(name) < schema.ValRepositoryMinNameLength {
		return fmt.Sprintf("Name must be at least %d characters long", schema.ValRepositoryMinNameLength), nil
	}
	if len(name) > schema.ValRepositoryMaxNameLength {
		return fmt.Sprintf("Name can not be longer than %d characters", schema.ValRepositoryMaxNameLength), nil
	}
	matched := schema.ValRepositoryNamePattern.MatchString(name)
	if !matched {
		return "Name can only contain letters, numbers, hyphens, and underscores", nil
	}

	exist, err := v.db.Repository.
		Query().
		Where(repository.Name(name)).
		Exist(v.ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Repository name must be unique", nil
	}

	return "", nil
}

func (v *ValidationClient) RepoPath(path string, isLocal bool) (string, error) {
	if path == "" {
		return "Path is required", nil
	}
	if isLocal {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~") {
			return "Path must start with / or ~", nil
		}
		if !v.backupClient().DoesPathExist(path) {
			return "Path does not exist", nil
		}
		if !v.backupClient().IsDirectory(path) {
			return "Path is not a folder", nil
		}
		if !v.backupClient().IsDirectoryEmpty(path) {
			// TODO: refactor this to have access to RepositoryService
			if !(*App)(v).RepositoryService().IsBorgRepository(path) {
				return "Folder must be empty", nil
			}
		}
	}

	exist, err := v.db.Repository.
		Query().
		Where(repository.Location(path)).
		Exist(v.ctx)
	if err != nil {
		return "", err
	}
	if exist {
		return "Repository is already connected", nil
	}

	return "", nil
}
