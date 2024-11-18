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

type WrappedExitErr struct {
	ExitError *exec.ExitError
}

func (e WrappedExitErr) Error() string {
	return e.ExitError.Error()
}

/***********************************/
/********** Borg Errors ************/
/***********************************/

type Error struct {
	WrappedExitErr
}

type ErrorWithTraceback struct {
	WrappedExitErr
}

type BufferMemoryLimitExceeded struct {
	WrappedExitErr
}

type EfficientCollectionQueueSizeUnderflow struct {
	WrappedExitErr
}

type RTError struct {
	WrappedExitErr
}

type CancelledByUser struct {
	WrappedExitErr
}

type CommandError struct {
	WrappedExitErr
}

type PlaceholderError struct {
	WrappedExitErr
}

type InvalidPlaceholder struct {
	WrappedExitErr
}

type RepositoryAlreadyExists struct {
	WrappedExitErr
}

type RepositoryAtticRepository struct {
	WrappedExitErr
}

type RepositoryCheckNeeded struct {
	WrappedExitErr
}

type RepositoryDoesNotExist struct {
	WrappedExitErr
}

type RepositoryInsufficientFreeSpaceError struct {
	WrappedExitErr
}

type RepositoryInvalidRepository struct {
	WrappedExitErr
}

type RepositoryInvalidRepositoryConfig struct {
	WrappedExitErr
}

type RepositoryObjectNotFound struct {
	WrappedExitErr
}

type RepositoryParentPathDoesNotExist struct {
	WrappedExitErr
}

type RepositoryPathAlreadyExists struct {
	WrappedExitErr
}

type RepositoryStorageQuotaExceeded struct {
	WrappedExitErr
}

type RepositoryPathPermissionDenied struct {
	WrappedExitErr
}

type MandatoryFeatureUnsupported struct {
	WrappedExitErr
}

type NoManifestError struct {
	WrappedExitErr
}

type UnsupportedManifestError struct {
	WrappedExitErr
}

type ArchiveAlreadyExists struct {
	WrappedExitErr
}

type ArchiveDoesNotExist struct {
	WrappedExitErr
}

type ArchiveIncompatibleFilesystemEncodingError struct {
	WrappedExitErr
}

type KeyfileInvalidError struct {
	WrappedExitErr
}

type KeyfileMismatchError struct {
	WrappedExitErr
}

type KeyfileNotFoundError struct {
	WrappedExitErr
}

type NotABorgKeyFile struct {
	WrappedExitErr
}

type RepoKeyNotFoundError struct {
	WrappedExitErr
}

type RepoIdMismatch struct {
	WrappedExitErr
}

type UnencryptedRepo struct {
	WrappedExitErr
}

type UnknownKeyType struct {
	WrappedExitErr
}

type UnsupportedPayloadError struct {
	WrappedExitErr
}

type NoPassphraseFailure struct {
	WrappedExitErr
}

type PasscommandFailure struct {
	WrappedExitErr
}

type PassphraseWrong struct {
	WrappedExitErr
}

type PasswordRetriesExceeded struct {
	WrappedExitErr
}

type CacheCacheInitAbortedError struct {
	WrappedExitErr
}

type CacheEncryptionMethodMismatch struct {
	WrappedExitErr
}

type CacheRepositoryAccessAborted struct {
	WrappedExitErr
}

type CacheRepositoryIDNotUnique struct {
	WrappedExitErr
}

type CacheRepositoryReplay struct {
	WrappedExitErr
}

type LockError struct {
	WrappedExitErr
}

type LockErrorT struct {
	WrappedExitErr
}

type LockFailed struct {
	WrappedExitErr
}

type LockTimeout struct {
	WrappedExitErr
}

type NotLocked struct {
	WrappedExitErr
}

type NotMyLock struct {
	WrappedExitErr
}

type ConnectionClosed struct {
	WrappedExitErr
}

type ConnectionClosedWithHint struct {
	WrappedExitErr
}

type InvalidRPCMethod struct {
	WrappedExitErr
}

type PathNotAllowed struct {
	WrappedExitErr
}

type RemoteRepositoryRPCServerOutdated struct {
	WrappedExitErr
}

type UnexpectedRPCDataFormatFromClient struct {
	WrappedExitErr
}

type UnexpectedRPCDataFormatFromServer struct {
	WrappedExitErr
}

type ConnectionBrokenWithHint struct {
	WrappedExitErr
}

type IntegrityError struct {
	WrappedExitErr
}

type FileIntegrityError struct {
	WrappedExitErr
}

type DecompressionError struct {
	WrappedExitErr
}

type ArchiveTAMInvalid struct {
	WrappedExitErr
}

type ArchiveTAMRequiredError struct {
	WrappedExitErr
}

type TAMInvalid struct {
	WrappedExitErr
}

type TAMRequiredError struct {
	WrappedExitErr
}

type TAMUnsupportedSuiteError struct {
	WrappedExitErr
}

type FileChangedWarning struct {
	WrappedExitErr
}

type IncludePatternNeverMatchedWarning struct {
	WrappedExitErr
}

type BackupError struct {
	WrappedExitErr
}

type BackupRaceConditionError struct {
	WrappedExitErr
}

type BackupOSError struct {
	WrappedExitErr
}

type BackupPermissionError struct {
	WrappedExitErr
}

type BackupIOError struct {
	WrappedExitErr
}

type BackupFileNotFoundError struct {
	WrappedExitErr
}

func (e Error) Error() string {
	return "Error rc: 2 traceback: no\nError: Error"
}

func (e ErrorWithTraceback) Error() string {
	return fmt.Sprintf("ErrorWithTraceback rc: 2 traceback: yes\nError: %s", e.WrappedExitErr.Error())
}

func (e BufferMemoryLimitExceeded) Error() string {
	return "Buffer.MemoryLimitExceeded rc: 2 traceback: no\nRequested buffer size is above the limit."
}

func (e EfficientCollectionQueueSizeUnderflow) Error() string {
	return "EfficientCollectionQueue.SizeUnderflow rc: 2 traceback: no\nCould not pop_front first elements, collection only has elements."
}

func (e RTError) Error() string {
	return "RTError rc: 2 traceback: no\nRuntime Error: Error"
}

func (e CancelledByUser) Error() string {
	return "CancelledByUser rc: 3 traceback: no\nCancelled by user."
}

func (e CommandError) Error() string {
	return "CommandError rc: 4 traceback: no\nCommand Error: Error"
}

func (e PlaceholderError) Error() string {
	return "PlaceholderError rc: 5 traceback: no\nFormatting Error: format: Error(Error)"
}

func (e InvalidPlaceholder) Error() string {
	return "InvalidPlaceholder rc: 6 traceback: no\nInvalid placeholder in string: Error"
}

func (e RepositoryAlreadyExists) Error() string {
	return "Repository.AlreadyExists rc: 10 traceback: no\nA repository already exists at location."
}

func (e RepositoryAtticRepository) Error() string {
	return "Repository.AtticRepository rc: 11 traceback: no\nAttic repository detected. Please run borg upgrade."
}

func (e RepositoryCheckNeeded) Error() string {
	return "Repository.CheckNeeded rc: 12 traceback: yes\nInconsistency detected. Please run borg check."
}

func (e RepositoryDoesNotExist) Error() string {
	return "Repository.DoesNotExist rc: 13 traceback: no\nRepository does not exist."
}

func (e RepositoryInsufficientFreeSpaceError) Error() string {
	return "Repository.InsufficientFreeSpaceError rc: 14 traceback: no\nInsufficient free space to complete transaction."
}

func (e RepositoryInvalidRepository) Error() string {
	return "Repository.InvalidRepository rc: 15 traceback: no\nInvalid repository. Check repo config."
}

func (e RepositoryInvalidRepositoryConfig) Error() string {
	return "Repository.InvalidRepositoryConfig rc: 16 traceback: no\nInvalid configuration. Check repo config."
}

func (e RepositoryObjectNotFound) Error() string {
	return "Repository.ObjectNotFound rc: 17 traceback: yes\nObject not found in repository."
}

func (e RepositoryParentPathDoesNotExist) Error() string {
	return "Repository.ParentPathDoesNotExist rc: 18 traceback: no\nThe parent path of the repo directory does not exist."
}

func (e RepositoryPathAlreadyExists) Error() string {
	return "Repository.PathAlreadyExists rc: 19 traceback: no\nThere is already something at path."
}

func (e RepositoryStorageQuotaExceeded) Error() string {
	return "Repository.StorageQuotaExceeded rc: 20 traceback: no\nThe storage quota has been exceeded. Try deleting some archives."
}

func (e RepositoryPathPermissionDenied) Error() string {
	return "Repository.PathPermissionDenied rc: 21 traceback: no\nPermission denied to path."
}

func (e MandatoryFeatureUnsupported) Error() string {
	return "MandatoryFeatureUnsupported rc: 25 traceback: no\nUnsupported repository feature(s). A newer version of borg is required to access this repository."
}

func (e NoManifestError) Error() string {
	return "NoManifestError rc: 26 traceback: no\nRepository has no manifest."
}

func (e UnsupportedManifestError) Error() string {
	return "UnsupportedManifestError rc: 27 traceback: no\nUnsupported manifest envelope. A newer version is required to access this repository."
}

func (e ArchiveAlreadyExists) Error() string {
	return "Archive.AlreadyExists rc: 30 traceback: no\nArchive already exists."
}

func (e ArchiveDoesNotExist) Error() string {
	return "Archive.DoesNotExist rc: 31 traceback: no\nArchive does not exist."
}

func (e ArchiveIncompatibleFilesystemEncodingError) Error() string {
	return "Archive.IncompatibleFilesystemEncodingError rc: 32 traceback: no\nFailed to encode filename into file system encoding. Consider configuring the LANG environment variable."
}

func (e KeyfileInvalidError) Error() string {
	return "KeyfileInvalidError rc: 40 traceback: no\nInvalid key data for repository found."
}

func (e KeyfileMismatchError) Error() string {
	return "KeyfileMismatchError rc: 41 traceback: no\nMismatch between repository and key file."
}

func (e KeyfileNotFoundError) Error() string {
	return "KeyfileNotFoundError rc: 42 traceback: no\nNo key file for repository found."
}

func (e NotABorgKeyFile) Error() string {
	return "NotABorgKeyFile rc: 43 traceback: no\nThis file is not a borg key backup, aborting."
}

func (e RepoKeyNotFoundError) Error() string {
	return "RepoKeyNotFoundError rc: 44 traceback: no\nNo key entry found in the config of repository."
}

func (e RepoIdMismatch) Error() string {
	return "RepoIdMismatch rc: 45 traceback: no\nThis key backup seems to be for a different backup repository, aborting."
}

func (e UnencryptedRepo) Error() string {
	return "UnencryptedRepo rc: 46 traceback: no\nKey management not available for unencrypted repositories."
}

func (e UnknownKeyType) Error() string {
	return "UnknownKeyType rc: 47 traceback: no\nKey type is unknown."
}

func (e UnsupportedPayloadError) Error() string {
	return "UnsupportedPayloadError rc: 48 traceback: no\nUnsupported payload type. A newer version is required to access this repository."
}

func (e NoPassphraseFailure) Error() string {
	return "NoPassphraseFailure rc: 50 traceback: no\nCannot acquire a passphrase."
}

func (e PasscommandFailure) Error() string {
	return "PasscommandFailure rc: 51 traceback: no\nPasscommand supplied in BORG_PASSCOMMAND failed."
}

func (e PassphraseWrong) Error() string {
	return "PassphraseWrong rc: 52 traceback: no\nPassphrase supplied in BORG_PASSPHRASE, by BORG_PASSCOMMAND or via BORG_PASSPHRASE_FD is incorrect."
}

func (e PasswordRetriesExceeded) Error() string {
	return "PasswordRetriesExceeded rc: 53 traceback: no\nExceeded the maximum password retries."
}

func (e CacheCacheInitAbortedError) Error() string {
	return "Cache.CacheInitAbortedError rc: 60 traceback: no\nCache initialization aborted."
}

func (e CacheEncryptionMethodMismatch) Error() string {
	return "Cache.EncryptionMethodMismatch rc: 61 traceback: no\nRepository encryption method changed since last access, refusing to continue."
}

func (e CacheRepositoryAccessAborted) Error() string {
	return "Cache.RepositoryAccessAborted rc: 62 traceback: no\nRepository access aborted."
}

func (e CacheRepositoryIDNotUnique) Error() string {
	return "Cache.RepositoryIDNotUnique rc: 63 traceback: no\nCache is newer than repository - do you have multiple, independently updated repos with same ID?"
}

func (e CacheRepositoryReplay) Error() string {
	return "Cache.RepositoryReplay rc: 64 traceback: no\nCache, or information obtained from the security directory is newer than repository - this is either an attack or unsafe (multiple repos with same ID)."
}

func (e LockError) Error() string {
	return "LockError rc: 70 traceback: no\nFailed to acquire the lock."
}

func (e LockErrorT) Error() string {
	return "LockErrorT rc: 71 traceback: yes\nFailed to acquire the lock."
}

func (e LockFailed) Error() string {
	return "LockFailed rc: 72 traceback: yes\nFailed to create/acquire the lock."
}

func (e LockTimeout) Error() string {
	return "LockTimeout rc: 73 traceback: no\nFailed to create/acquire the lock (timeout)."
}

func (e NotLocked) Error() string {
	return "NotLocked rc: 74 traceback: yes\nFailed to release the lock (was not locked)."
}

func (e NotMyLock) Error() string {
	return "NotMyLock rc: 75 traceback: yes\nFailed to release the lock (was/is locked, but not by me)."
}

func (e ConnectionClosed) Error() string {
	return "ConnectionClosed rc: 80 traceback: no\nConnection closed by remote host."
}

func (e ConnectionClosedWithHint) Error() string {
	return "ConnectionClosedWithHint rc: 81 traceback: no\nConnection closed by remote host."
}

func (e InvalidRPCMethod) Error() string {
	return "InvalidRPCMethod rc: 82 traceback: no\nRPC method is not valid."
}

func (e PathNotAllowed) Error() string {
	return "PathNotAllowed rc: 83 traceback: no\nRepository path not allowed."
}

func (e RemoteRepositoryRPCServerOutdated) Error() string {
	return "RemoteRepository.RPCServerOutdated rc: 84 traceback: no\nBorg server is too old. Required version."
}

func (e UnexpectedRPCDataFormatFromClient) Error() string {
	return "UnexpectedRPCDataFormatFromClient rc: 85 traceback: no\nGot unexpected RPC data format from client."
}

func (e UnexpectedRPCDataFormatFromServer) Error() string {
	return "UnexpectedRPCDataFormatFromServer rc: 86 traceback: no\nGot unexpected RPC data format from server."
}

func (e ConnectionBrokenWithHint) Error() string {
	return "ConnectionBrokenWithHint rc: 87 traceback: no\nConnection to remote host is broken."
}

func (e IntegrityError) Error() string {
	return "IntegrityError rc: 90 traceback: yes\nData integrity error."
}

func (e FileIntegrityError) Error() string {
	return "FileIntegrityError rc: 91 traceback: yes\nFile failed integrity check."
}

func (e DecompressionError) Error() string {
	return "DecompressionError rc: 92 traceback: yes\nDecompression error."
}

func (e ArchiveTAMInvalid) Error() string {
	return "ArchiveTAMInvalid rc: 95 traceback: yes\nData integrity error."
}

func (e ArchiveTAMRequiredError) Error() string {
	return "ArchiveTAMRequiredError rc: 96 traceback: yes\nArchive is unauthenticated, but it is required for this repository."
}

func (e TAMInvalid) Error() string {
	return "TAMInvalid rc: 97 traceback: yes\nData integrity error."
}

func (e TAMRequiredError) Error() string {
	return "TAMRequiredError rc: 98 traceback: yes\nManifest is unauthenticated, but it is required for this repository."
}

func (e TAMUnsupportedSuiteError) Error() string {
	return "TAMUnsupportedSuiteError rc: 99 traceback: yes\nCould not verify manifest: Unsupported suite; a newer version is needed."
}

func (e FileChangedWarning) Error() string {
	return "FileChangedWarning rc: 100\nFile changed while we backed it up."
}

func (e IncludePatternNeverMatchedWarning) Error() string {
	return "IncludePatternNeverMatchedWarning rc: 101\nInclude pattern never matched."
}

func (e BackupError) Error() string {
	return "BackupError rc: 102\nBackup error."
}

func (e BackupRaceConditionError) Error() string {
	return "BackupRaceConditionError rc: 103\nFile type or inode changed while we backed it up (race condition, skipped file)."
}

func (e BackupOSError) Error() string {
	return "BackupOSError rc: 104\nBackup OS error."
}

func (e BackupPermissionError) Error() string {
	return "BackupPermissionError rc: 105\nBackup permission error."
}

func (e BackupIOError) Error() string {
	return "BackupIOError rc: 106\nBackup IO error."
}

func (e BackupFileNotFoundError) Error() string {
	return "BackupFileNotFoundError rc: 107\nBackup file not found."
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
		return Error{WrappedExitErr{ExitError: exitError}}
	case 3:
		return CancelledByUser{WrappedExitErr{ExitError: exitError}}
	case 4:
		return CommandError{WrappedExitErr{ExitError: exitError}}
	case 5:
		return PlaceholderError{WrappedExitErr{ExitError: exitError}}
	case 6:
		return InvalidPlaceholder{WrappedExitErr{ExitError: exitError}}
	case 10:
		return RepositoryAlreadyExists{WrappedExitErr{ExitError: exitError}}
	case 11:
		return RepositoryAtticRepository{WrappedExitErr{ExitError: exitError}}
	case 12:
		return RepositoryCheckNeeded{WrappedExitErr{ExitError: exitError}}
	case 13:
		return RepositoryDoesNotExist{WrappedExitErr{ExitError: exitError}}
	case 14:
		return RepositoryInsufficientFreeSpaceError{WrappedExitErr{ExitError: exitError}}
	case 15:
		return RepositoryInvalidRepository{WrappedExitErr{ExitError: exitError}}
	case 16:
		return RepositoryInvalidRepositoryConfig{WrappedExitErr{ExitError: exitError}}
	case 17:
		return RepositoryObjectNotFound{WrappedExitErr{ExitError: exitError}}
	case 18:
		return RepositoryParentPathDoesNotExist{WrappedExitErr{ExitError: exitError}}
	case 19:
		return RepositoryPathAlreadyExists{WrappedExitErr{ExitError: exitError}}
	case 20:
		return RepositoryStorageQuotaExceeded{WrappedExitErr{ExitError: exitError}}
	case 21:
		return RepositoryPathPermissionDenied{WrappedExitErr{ExitError: exitError}}
	case 25:
		return MandatoryFeatureUnsupported{WrappedExitErr{ExitError: exitError}}
	case 26:
		return NoManifestError{WrappedExitErr{ExitError: exitError}}
	case 27:
		return UnsupportedManifestError{WrappedExitErr{ExitError: exitError}}
	case 30:
		return ArchiveAlreadyExists{WrappedExitErr{ExitError: exitError}}
	case 31:
		return ArchiveDoesNotExist{WrappedExitErr{ExitError: exitError}}
	case 32:
		return ArchiveIncompatibleFilesystemEncodingError{WrappedExitErr{ExitError: exitError}}
	case 40:
		return KeyfileInvalidError{WrappedExitErr{ExitError: exitError}}
	case 41:
		return KeyfileMismatchError{WrappedExitErr{ExitError: exitError}}
	case 42:
		return KeyfileNotFoundError{WrappedExitErr{ExitError: exitError}}
	case 43:
		return NotABorgKeyFile{WrappedExitErr{ExitError: exitError}}
	case 44:
		return RepoKeyNotFoundError{WrappedExitErr{ExitError: exitError}}
	case 45:
		return RepoIdMismatch{WrappedExitErr{ExitError: exitError}}
	case 46:
		return UnencryptedRepo{WrappedExitErr{ExitError: exitError}}
	case 47:
		return UnknownKeyType{WrappedExitErr{ExitError: exitError}}
	case 48:
		return UnsupportedPayloadError{WrappedExitErr{ExitError: exitError}}
	case 50:
		return NoPassphraseFailure{WrappedExitErr{ExitError: exitError}}
	case 51:
		return PasscommandFailure{WrappedExitErr{ExitError: exitError}}
	case 52:
		return PassphraseWrong{WrappedExitErr{ExitError: exitError}}
	case 53:
		return PasswordRetriesExceeded{WrappedExitErr{ExitError: exitError}}
	case 60:
		return CacheCacheInitAbortedError{WrappedExitErr{ExitError: exitError}}
	case 61:
		return CacheEncryptionMethodMismatch{WrappedExitErr{ExitError: exitError}}
	case 62:
		return CacheRepositoryAccessAborted{WrappedExitErr{ExitError: exitError}}
	case 63:
		return CacheRepositoryIDNotUnique{WrappedExitErr{ExitError: exitError}}
	case 64:
		return CacheRepositoryReplay{WrappedExitErr{ExitError: exitError}}
	case 70:
		return LockError{WrappedExitErr{ExitError: exitError}}
	case 71:
		return LockErrorT{WrappedExitErr{ExitError: exitError}}
	case 72:
		return LockFailed{WrappedExitErr{ExitError: exitError}}
	case 73:
		return LockTimeout{WrappedExitErr{ExitError: exitError}}
	case 74:
		return NotLocked{WrappedExitErr{ExitError: exitError}}
	case 75:
		return NotMyLock{WrappedExitErr{ExitError: exitError}}
	case 80:
		return ConnectionClosed{WrappedExitErr{ExitError: exitError}}
	case 81:
		return ConnectionClosedWithHint{WrappedExitErr{ExitError: exitError}}
	case 82:
		return InvalidRPCMethod{WrappedExitErr{ExitError: exitError}}
	case 83:
		return PathNotAllowed{WrappedExitErr{ExitError: exitError}}
	case 84:
		return RemoteRepositoryRPCServerOutdated{WrappedExitErr{ExitError: exitError}}
	case 85:
		return UnexpectedRPCDataFormatFromClient{WrappedExitErr{ExitError: exitError}}
	case 86:
		return UnexpectedRPCDataFormatFromServer{WrappedExitErr{ExitError: exitError}}
	case 87:
		return ConnectionBrokenWithHint{WrappedExitErr{ExitError: exitError}}
	case 90:
		return IntegrityError{WrappedExitErr{ExitError: exitError}}
	case 91:
		return FileIntegrityError{WrappedExitErr{ExitError: exitError}}
	case 92:
		return DecompressionError{WrappedExitErr{ExitError: exitError}}
	case 95:
		return ArchiveTAMInvalid{WrappedExitErr{ExitError: exitError}}
	case 96:
		return ArchiveTAMRequiredError{WrappedExitErr{ExitError: exitError}}
	case 97:
		return TAMInvalid{WrappedExitErr{ExitError: exitError}}
	case 98:
		return TAMRequiredError{WrappedExitErr{ExitError: exitError}}
	case 99:
		return TAMUnsupportedSuiteError{WrappedExitErr{ExitError: exitError}}
	case 100:
		return FileChangedWarning{WrappedExitErr{ExitError: exitError}}
	case 101:
		return IncludePatternNeverMatchedWarning{WrappedExitErr{ExitError: exitError}}
	case 102:
		return BackupError{WrappedExitErr{ExitError: exitError}}
	case 103:
		return BackupRaceConditionError{WrappedExitErr{ExitError: exitError}}
	case 104:
		return BackupOSError{WrappedExitErr{ExitError: exitError}}
	case 105:
		return BackupPermissionError{WrappedExitErr{ExitError: exitError}}
	case 106:
		return BackupIOError{WrappedExitErr{ExitError: exitError}}
	case 107:
		return BackupFileNotFoundError{WrappedExitErr{ExitError: exitError}}
	default:
		return err
	}
}
