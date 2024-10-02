package borg

import (
	"errors"
	"fmt"
	"os/exec"
)

/***********************************/
/********** Self Defined ***********/
/***********************************/

type CancelErr struct{}

func (CancelErr) Error() string {
	return "command canceled"
}

/***********************************/
/********** borg Errors ************/
/***********************************/

type Error struct{}

func (Error) Error() string { return "Error: {}" }

type ErrorWithTraceback struct{}

func (ErrorWithTraceback) Error() string { return "Error: {}" }

type BufferMemoryLimitExceeded struct {
	RequestedSize int
	Limit         int
}

func (e BufferMemoryLimitExceeded) Error() string {
	return fmt.Sprintf("Requested buffer size %d is above the limit of %d.", e.RequestedSize, e.Limit)
}

type EfficientCollectionQueueSizeUnderflow struct {
	FirstElements  int
	CollectionSize int
}

func (e EfficientCollectionQueueSizeUnderflow) Error() string {
	return fmt.Sprintf("Could not pop_front first %d elements, collection only has %d elements.", e.FirstElements, e.CollectionSize)
}

type RTError struct{}

func (RTError) Error() string { return "Runtime Error: {}" }

type CancelledByUser struct{}

func (CancelledByUser) Error() string { return "Cancelled by user." }

type CommandError struct{}

func (CommandError) Error() string { return "Command Error: {}" }

type PlaceholderError struct {
	Format string
	Args   []interface{}
}

func (e PlaceholderError) Error() string {
	return fmt.Sprintf("Formatting Error: “%s”.format(%v): %v", e.Format, e.Args[0], e.Args[1])
}

type InvalidPlaceholder struct {
	Placeholder string
}

func (e InvalidPlaceholder) Error() string {
	return fmt.Sprintf("Invalid placeholder “%s” in string: {}", e.Placeholder)
}

type RepositoryAlreadyExists struct {
	Path string
}

func (e RepositoryAlreadyExists) Error() string {
	return fmt.Sprintf("A repository already exists at %s.", e.Path)
}

type RepositoryAtticRepository struct {
	Path string
}

func (e RepositoryAtticRepository) Error() string {
	return fmt.Sprintf("Attic repository detected. Please run “borg upgrade %s”.", e.Path)
}

type RepositoryCheckNeeded struct {
	Path string
}

func (e RepositoryCheckNeeded) Error() string {
	return fmt.Sprintf("Inconsistency detected. Please run “borg check %s”.", e.Path)
}

type RepositoryDoesNotExist struct {
	Path string
}

func (e RepositoryDoesNotExist) Error() string {
	return fmt.Sprintf("Repository %s does not exist.", e.Path)
}

type RepositoryInsufficientFreeSpaceError struct {
	Required  int
	Available int
}

func (e RepositoryInsufficientFreeSpaceError) Error() string {
	return fmt.Sprintf("Insufficient free space to complete transaction (required: %d, available: %d).", e.Required, e.Available)
}

type RepositoryInvalidRepository struct {
	Path string
}

func (e RepositoryInvalidRepository) Error() string {
	return fmt.Sprintf("%s is not a valid repository. Check repo config.", e.Path)
}

type RepositoryInvalidRepositoryConfig struct {
	Path   string
	Config string
}

func (e RepositoryInvalidRepositoryConfig) Error() string {
	return fmt.Sprintf("%s does not have a valid configuration. Check repo config [%s].", e.Path, e.Config)
}

type RepositoryObjectNotFound struct {
	Key  string
	Repo string
}

func (e RepositoryObjectNotFound) Error() string {
	return fmt.Sprintf("Object with key %s not found in repository %s.", e.Key, e.Repo)
}

type RepositoryParentPathDoesNotExist struct {
	Path string
}

func (e RepositoryParentPathDoesNotExist) Error() string {
	return fmt.Sprintf("The parent path of the repo directory [%s] does not exist.", e.Path)
}

type RepositoryPathAlreadyExists struct {
	Path string
}

func (e RepositoryPathAlreadyExists) Error() string {
	return fmt.Sprintf("There is already something at %s.", e.Path)
}

type RepositoryStorageQuotaExceeded struct {
	Quota    string
	Exceeded string
}

func (e RepositoryStorageQuotaExceeded) Error() string {
	return fmt.Sprintf("The storage quota (%s) has been exceeded (%s). Try deleting some archives.", e.Quota, e.Exceeded)
}

type RepositoryPathPermissionDenied struct {
	Path string
}

func (e RepositoryPathPermissionDenied) Error() string {
	return fmt.Sprintf("Permission denied to %s.", e.Path)
}

type MandatoryFeatureUnsupported struct {
	Features string
}

func (e MandatoryFeatureUnsupported) Error() string {
	return fmt.Sprintf("Unsupported repository feature(s) %s. A newer version of borg is required to access this repository.", e.Features)
}

type NoManifestError struct{}

func (NoManifestError) Error() string { return "Repository has no manifest." }

type UnsupportedManifestError struct{}

func (UnsupportedManifestError) Error() string {
	return "Unsupported manifest envelope. A newer version is required to access this repository."
}

type ArchiveAlreadyExists struct {
	Name string
}

func (e ArchiveAlreadyExists) Error() string {
	return fmt.Sprintf("Archive %s already exists", e.Name)
}

type ArchiveDoesNotExist struct {
	Name string
}

func (e ArchiveDoesNotExist) Error() string {
	return fmt.Sprintf("Archive %s does not exist", e.Name)
}

type ArchiveIncompatibleFilesystemEncodingError struct {
	Filename string
	Encoding string
}

func (e ArchiveIncompatibleFilesystemEncodingError) Error() string {
	return fmt.Sprintf("Failed to encode filename “%s” into file system encoding “%s”. Consider configuring the LANG environment variable.", e.Filename, e.Encoding)
}

type KeyfileInvalidError struct {
	Repo string
	Path string
}

func (e KeyfileInvalidError) Error() string {
	return fmt.Sprintf("Invalid key data for repository %s found in %s.", e.Repo, e.Path)
}

type KeyfileMismatchError struct {
	Repo string
	Path string
}

func (e KeyfileMismatchError) Error() string {
	return fmt.Sprintf("Mismatch between repository %s and key file %s.", e.Repo, e.Path)
}

type KeyfileNotFoundError struct {
	Repo string
	Path string
}

func (e KeyfileNotFoundError) Error() string {
	return fmt.Sprintf("No key file for repository %s found in %s.", e.Repo, e.Path)
}

type NotABorgKeyFile struct{}

func (NotABorgKeyFile) Error() string { return "This file is not a borg key backup, aborting." }

type RepoKeyNotFoundError struct {
	Repo string
}

func (e RepoKeyNotFoundError) Error() string {
	return fmt.Sprintf("No key entry found in the config of repository %s.", e.Repo)
}

type RepoIdMismatch struct{}

func (RepoIdMismatch) Error() string {
	return "This key backup seems to be for a different backup repository, aborting."
}

type UnencryptedRepo struct{}

func (UnencryptedRepo) Error() string {
	return "Key management not available for unencrypted repositories."
}

type UnknownKeyType struct {
	Type string
}

func (e UnknownKeyType) Error() string {
	return fmt.Sprintf("Key type %s is unknown.", e.Type)
}

type UnsupportedPayloadError struct {
	Type string
}

func (e UnsupportedPayloadError) Error() string {
	return fmt.Sprintf("Unsupported payload type %s. A newer version is required to access this repository.", e.Type)
}

type NoPassphraseFailure struct {
	Reason string
}

func (e NoPassphraseFailure) Error() string {
	return fmt.Sprintf("can not acquire a passphrase: %s", e.Reason)
}

type PasscommandFailure struct{}

func (PasscommandFailure) Error() string {
	return "passcommand supplied in BORG_PASSCOMMAND failed: {}"
}

type PassphraseWrong struct{}

func (PassphraseWrong) Error() string {
	return "passphrase supplied in BORG_PASSPHRASE, by BORG_PASSCOMMAND or via BORG_PASSPHRASE_FD is incorrect."
}

type PasswordRetriesExceeded struct{}

func (PasswordRetriesExceeded) Error() string { return "exceeded the maximum password retries" }

type CacheCacheInitAbortedError struct{}

func (CacheCacheInitAbortedError) Error() string { return "Cache initialization aborted" }

type CacheEncryptionMethodMismatch struct{}

func (CacheEncryptionMethodMismatch) Error() string {
	return "Repository encryption method changed since last access, refusing to continue"
}

type CacheRepositoryAccessAborted struct{}

func (CacheRepositoryAccessAborted) Error() string { return "Repository access aborted" }

type CacheRepositoryIDNotUnique struct{}

func (CacheRepositoryIDNotUnique) Error() string {
	return "Cache is newer than repository - do you have multiple, independently updated repos with same ID?"
}

type CacheRepositoryReplay struct{}

func (CacheRepositoryReplay) Error() string {
	return "Cache, or information obtained from the security directory is newer than repository - this is either an attack or unsafe (multiple repos with same ID)"
}

type LockError struct {
	Lock string
}

func (e LockError) Error() string {
	return fmt.Sprintf("Failed to acquire the lock %s.", e.Lock)
}

type LockErrorT struct {
	Lock string
}

func (e LockErrorT) Error() string {
	return fmt.Sprintf("Failed to acquire the lock %s.", e.Lock)
}

type LockFailed struct {
	Lock   string
	Reason string
}

func (e LockFailed) Error() string {
	return fmt.Sprintf("Failed to create/acquire the lock %s (%s).", e.Lock, e.Reason)
}

type LockTimeout struct {
	Lock string
}

func (e LockTimeout) Error() string {
	return fmt.Sprintf("Failed to create/acquire the lock %s (timeout).", e.Lock)
}

type NotLocked struct {
	Lock string
}

func (e NotLocked) Error() string {
	return fmt.Sprintf("Failed to release the lock %s (was not locked).", e.Lock)
}

type NotMyLock struct {
	Lock string
}

func (e NotMyLock) Error() string {
	return fmt.Sprintf("Failed to release the lock %s (was/is locked, but not by me).", e.Lock)
}

type ConnectionClosed struct{}

func (ConnectionClosed) Error() string { return "Connection closed by remote host" }

type ConnectionClosedWithHint struct {
	Hint string
}

func (e ConnectionClosedWithHint) Error() string {
	return fmt.Sprintf("Connection closed by remote host. %s", e.Hint)
}

type InvalidRPCMethod struct {
	Method string
}

func (e InvalidRPCMethod) Error() string {
	return fmt.Sprintf("RPC method %s is not valid", e.Method)
}

type PathNotAllowed struct {
	Path string
}

func (e PathNotAllowed) Error() string {
	return fmt.Sprintf("Repository path not allowed: %s", e.Path)
}

type RemoteRepositoryRPCServerOutdated struct {
	Version         string
	RequiredVersion string
}

func (e RemoteRepositoryRPCServerOutdated) Error() string {
	return fmt.Sprintf("borg server is too old for %s. Required version %s", e.Version, e.RequiredVersion)
}

type UnexpectedRPCDataFormatFromClient struct {
	Client string
}

func (e UnexpectedRPCDataFormatFromClient) Error() string {
	return fmt.Sprintf("borg %s: Got unexpected RPC data format from client.", e.Client)
}

type UnexpectedRPCDataFormatFromServer struct {
	Server string
}

func (e UnexpectedRPCDataFormatFromServer) Error() string {
	return fmt.Sprintf("Got unexpected RPC data format from server: %s", e.Server)
}

type ConnectionBrokenWithHint struct {
	Hint string
}

func (e ConnectionBrokenWithHint) Error() string {
	return fmt.Sprintf("Connection to remote host is broken. %s", e.Hint)
}

type IntegrityError struct{}

func (IntegrityError) Error() string { return "Data integrity error: {}" }

type FileIntegrityError struct{}

func (FileIntegrityError) Error() string { return "File failed integrity check: {}" }

type DecompressionError struct{}

func (DecompressionError) Error() string { return "Decompression error: {}" }

type ArchiveTAMInvalid struct{}

func (ArchiveTAMInvalid) Error() string { return "Data integrity error: {}" }

type ArchiveTAMRequiredError struct {
	Archive string
}

func (e ArchiveTAMRequiredError) Error() string {
	return fmt.Sprintf("Archive ‘%s’ is unauthenticated, but it is required for this repository.", e.Archive)
}

type TAMInvalid struct{}

func (TAMInvalid) Error() string { return "Data integrity error: {}" }

type TAMRequiredError struct{}

func (TAMRequiredError) Error() string {
	return "Manifest is unauthenticated, but it is required for this repository."
}

type TAMUnsupportedSuiteError struct {
	Suite string
}

func (e TAMUnsupportedSuiteError) Error() string {
	return fmt.Sprintf("Could not verify manifest: Unsupported suite %s; a newer version is needed.", e.Suite)
}

func exitCodesToError(err error) error {
	// Return nil if there is no error
	if err == nil {
		return nil
	}

	// Return the error if it is not an ExitError
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return err
	}

	// Return the error based on the exit code
	switch exitError.ExitCode() {
	case 2:
		return Error{}
	case 3:
		return CancelledByUser{}
	case 4:
		return CommandError{}
	case 5:
		return PlaceholderError{}
	case 6:
		return InvalidPlaceholder{}
	case 10:
		return RepositoryAlreadyExists{}
	case 11:
		return RepositoryAtticRepository{}
	case 12:
		return RepositoryCheckNeeded{}
	case 13:
		return RepositoryDoesNotExist{}
	case 14:
		return RepositoryInsufficientFreeSpaceError{}
	case 15:
		return RepositoryInvalidRepository{}
	case 16:
		return RepositoryInvalidRepositoryConfig{}
	case 17:
		return RepositoryObjectNotFound{}
	case 18:
		return RepositoryParentPathDoesNotExist{}
	case 19:
		return RepositoryPathAlreadyExists{}
	case 20:
		return RepositoryStorageQuotaExceeded{}
	case 21:
		return RepositoryPathPermissionDenied{}
	case 25:
		return MandatoryFeatureUnsupported{}
	case 26:
		return NoManifestError{}
	case 27:
		return UnsupportedManifestError{}
	case 30:
		return ArchiveAlreadyExists{}
	case 31:
		return ArchiveDoesNotExist{}
	case 32:
		return ArchiveIncompatibleFilesystemEncodingError{}
	case 40:
		return KeyfileInvalidError{}
	case 41:
		return KeyfileMismatchError{}
	case 42:
		return KeyfileNotFoundError{}
	case 43:
		return NotABorgKeyFile{}
	case 44:
		return RepoKeyNotFoundError{}
	case 45:
		return RepoIdMismatch{}
	case 46:
		return UnencryptedRepo{}
	case 47:
		return UnknownKeyType{}
	case 48:
		return UnsupportedPayloadError{}
	case 50:
		return NoPassphraseFailure{}
	case 51:
		return PasscommandFailure{}
	case 52:
		return PassphraseWrong{}
	case 53:
		return PasswordRetriesExceeded{}
	case 60:
		return CacheCacheInitAbortedError{}
	case 61:
		return CacheEncryptionMethodMismatch{}
	case 62:
		return CacheRepositoryAccessAborted{}
	case 63:
		return CacheRepositoryIDNotUnique{}
	case 64:
		return CacheRepositoryReplay{}
	case 70:
		return LockError{}
	case 71:
		return LockErrorT{}
	case 72:
		return LockFailed{}
	case 73:
		return LockTimeout{}
	case 74:
		return NotLocked{}
	case 75:
		return NotMyLock{}
	case 80:
		return ConnectionClosed{}
	case 81:
		return ConnectionClosedWithHint{}
	case 82:
		return InvalidRPCMethod{}
	case 83:
		return PathNotAllowed{}
	case 84:
		return RemoteRepositoryRPCServerOutdated{}
	case 85:
		return UnexpectedRPCDataFormatFromClient{}
	case 86:
		return UnexpectedRPCDataFormatFromServer{}
	case 87:
		return ConnectionBrokenWithHint{}
	case 90:
		return IntegrityError{}
	case 91:
		return FileIntegrityError{}
	case 92:
		return DecompressionError{}
	case 95:
		return ArchiveTAMInvalid{}
	case 96:
		return ArchiveTAMRequiredError{}
	case 97:
		return TAMInvalid{}
	case 98:
		return TAMRequiredError{}
	case 99:
		return TAMUnsupportedSuiteError{}
	default:
		return err
	}
}
