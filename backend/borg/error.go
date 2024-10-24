package borg

import (
	"errors"
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

func (e BackupPermissionError) Error() string {
	return "permission error"
}

type BackupIOError struct {
	WrappedExitErr
}

type BackupFileNotFoundError struct {
	WrappedExitErr
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
