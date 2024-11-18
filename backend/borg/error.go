package borg

import (
	"errors"
	"github.com/negrel/assert"
	"os/exec"
)

/***********************************/
/********** Self Defined ***********/
/***********************************/

type CancelErr struct{}

func (CancelErr) Error() string {
	return "command canceled"
}

type WithExitError struct {
	ExitError *exec.ExitError
}

func (e WithExitError) Error() string {
	return e.ExitError.Error()
}

type Error struct {
	WithExitError
}

/***********************************/
/********** Borg Errors ************/
/***********************************/

var (
	ErrDefault                                 = errors.New("error")
	ErrWithTraceback                           = errors.New("error with traceback")
	ErrorBufferMemoryLimitExceeded             = errors.New("buffer memory limit exceeded")
	ErrorEfficientCollectionQueueSizeUnderflow = errors.New("efficient collection queue size underflow")
	ErrorRTError                               = errors.New("runtime error")
	ErrorCancelledByUser                       = errors.New("cancelled by user")
	ErrorCommandError                          = errors.New("command error")
	ErrorPlaceholderError                      = errors.New("placeholder error")
	ErrorInvalidPlaceholder                    = errors.New("invalid placeholder")
	ErrorRepositoryAlreadyExists               = errors.New("repository already exists")
	ErrorRepositoryAtticRepository             = errors.New("attic repository detected")
	ErrorRepositoryCheckNeeded                 = errors.New("repository check needed")
	ErrorRepositoryDoesNotExist                = errors.New("repository does not exist")
	ErrorRepositoryInsufficientFreeSpaceError  = errors.New("insufficient free space")
	ErrorRepositoryInvalidRepository           = errors.New("invalid repository")
	ErrorRepositoryInvalidRepositoryConfig     = errors.New("invalid repository config")
	ErrorRepositoryObjectNotFound              = errors.New("object not found in repository")
	ErrorRepositoryParentPathDoesNotExist      = errors.New("parent path does not exist")
	ErrorRepositoryPathAlreadyExists           = errors.New("path already exists")
	ErrorRepositoryStorageQuotaExceeded        = errors.New("storage quota exceeded")
	ErrorRepositoryPathPermissionDenied        = errors.New("permission denied to path")
	ErrorMandatoryFeatureUnsupported           = errors.New("unsupported repository feature")
	ErrorNoManifestError                       = errors.New("repository has no manifest")
	ErrorUnsupportedManifestError              = errors.New("unsupported manifest envelope")
	ErrorArchiveAlreadyExists                  = errors.New("archive already exists")
	ErrorArchiveDoesNotExist                   = errors.New("archive does not exist")
	ErrorArchiveIncompatibleFilesystemEncoding = errors.New("failed to encode filename")
	ErrorKeyfileInvalidError                   = errors.New("invalid key data")
	ErrorKeyfileMismatchError                  = errors.New("mismatch between repository and key file")
	ErrorKeyfileNotFoundError                  = errors.New("no key file found")
	ErrorNotABorgKeyFile                       = errors.New("not a borg key backup")
	ErrorRepoKeyNotFoundError                  = errors.New("no key entry found")
	ErrorRepoIdMismatch                        = errors.New("key backup for different repository")
	ErrorUnencryptedRepo                       = errors.New("key management not available")
	ErrorUnknownKeyType                        = errors.New("unknown key type")
	ErrorUnsupportedPayloadError               = errors.New("unsupported payload type")
	ErrorNoPassphraseFailure                   = errors.New("cannot acquire a passphrase")
	ErrorPasscommandFailure                    = errors.New("passcommand failed")
	ErrorPassphraseWrong                       = errors.New("incorrect passphrase")
	ErrorPasswordRetriesExceeded               = errors.New("exceeded password retries")
	ErrorCacheInitAborted                      = errors.New("cache initialization aborted")
	ErrorCacheEncryptionMethodMismatch         = errors.New("encryption method mismatch")
	ErrorCacheRepositoryAccessAborted          = errors.New("repository access aborted")
	ErrorCacheRepositoryIDNotUnique            = errors.New("repository ID not unique")
	ErrorCacheRepositoryReplay                 = errors.New("cache newer than repository")
	ErrorLockError                             = errors.New("failed to acquire lock")
	ErrorLockErrorT                            = errors.New("failed to acquire lock with traceback")
	ErrorLockFailed                            = errors.New("failed to create/acquire lock")
	ErrorLockTimeout                           = errors.New("lock timeout")
	ErrorNotLocked                             = errors.New("failed to release lock (not locked)")
	ErrorNotMyLock                             = errors.New("failed to release lock (not by me)")
	ErrorConnectionClosed                      = errors.New("connection closed by remote host")
	ErrorConnectionClosedWithHint              = errors.New("connection closed by remote host with hint")
	ErrorInvalidRPCMethod                      = errors.New("invalid RPC method")
	ErrorPathNotAllowed                        = errors.New("repository path not allowed")
	ErrorRemoteRepositoryRPCServerOutdated     = errors.New("borg server too old")
	ErrorUnexpectedRPCDataFormatFromClient     = errors.New("unexpected RPC data format from client")
	ErrorUnexpectedRPCDataFormatFromServer     = errors.New("unexpected RPC data format from server")
	ErrorConnectionBrokenWithHint              = errors.New("connection to remote host broken")
	ErrorIntegrityError                        = errors.New("data integrity error")
	ErrorFileIntegrityError                    = errors.New("file integrity check failed")
	ErrorDecompressionError                    = errors.New("decompression error")
	ErrorArchiveTAMInvalid                     = errors.New("archive TAM invalid")
	ErrorArchiveTAMRequiredError               = errors.New("archive unauthenticated")
	ErrorTAMInvalid                            = errors.New("TAM invalid")
	ErrorTAMRequiredError                      = errors.New("manifest unauthenticated")
	ErrorTAMUnsupportedSuiteError              = errors.New("unsupported suite")
	ErrorFileChangedWarning                    = errors.New("file changed during backup")
	ErrorIncludePatternNeverMatchedWarning     = errors.New("include pattern never matched")
	ErrorBackupError                           = errors.New("backup error")
	ErrorBackupRaceConditionError              = errors.New("file type or inode changed during backup")
	ErrorBackupOSError                         = errors.New("backup OS error")
	ErrorBackupPermissionError                 = errors.New("backup permission error")
	ErrorBackupIOError                         = errors.New("backup IO error")
	ErrorBackupFileNotFoundError               = errors.New("backup file not found")
)

var errorMap = map[int]error{
	2:   ErrDefault,
	3:   ErrorCancelledByUser,
	4:   ErrorCommandError,
	5:   ErrorPlaceholderError,
	6:   ErrorInvalidPlaceholder,
	10:  ErrorRepositoryAlreadyExists,
	11:  ErrorRepositoryAtticRepository,
	12:  ErrorRepositoryCheckNeeded,
	13:  ErrorRepositoryDoesNotExist,
	14:  ErrorRepositoryInsufficientFreeSpaceError,
	15:  ErrorRepositoryInvalidRepository,
	16:  ErrorRepositoryInvalidRepositoryConfig,
	17:  ErrorRepositoryObjectNotFound,
	18:  ErrorRepositoryParentPathDoesNotExist,
	19:  ErrorRepositoryPathAlreadyExists,
	20:  ErrorRepositoryStorageQuotaExceeded,
	21:  ErrorRepositoryPathPermissionDenied,
	25:  ErrorMandatoryFeatureUnsupported,
	26:  ErrorNoManifestError,
	27:  ErrorUnsupportedManifestError,
	30:  ErrorArchiveAlreadyExists,
	31:  ErrorArchiveDoesNotExist,
	32:  ErrorArchiveIncompatibleFilesystemEncoding,
	40:  ErrorKeyfileInvalidError,
	41:  ErrorKeyfileMismatchError,
	42:  ErrorKeyfileNotFoundError,
	43:  ErrorNotABorgKeyFile,
	44:  ErrorRepoKeyNotFoundError,
	45:  ErrorRepoIdMismatch,
	46:  ErrorUnencryptedRepo,
	47:  ErrorUnknownKeyType,
	48:  ErrorUnsupportedPayloadError,
	50:  ErrorNoPassphraseFailure,
	51:  ErrorPasscommandFailure,
	52:  ErrorPassphraseWrong,
	53:  ErrorPasswordRetriesExceeded,
	60:  ErrorCacheInitAborted,
	61:  ErrorCacheEncryptionMethodMismatch,
	62:  ErrorCacheRepositoryAccessAborted,
	63:  ErrorCacheRepositoryIDNotUnique,
	64:  ErrorCacheRepositoryReplay,
	70:  ErrorLockError,
	71:  ErrorLockErrorT,
	72:  ErrorLockFailed,
	73:  ErrorLockTimeout,
	74:  ErrorNotLocked,
	75:  ErrorNotMyLock,
	80:  ErrorConnectionClosed,
	81:  ErrorConnectionClosedWithHint,
	82:  ErrorInvalidRPCMethod,
	83:  ErrorPathNotAllowed,
	84:  ErrorRemoteRepositoryRPCServerOutdated,
	85:  ErrorUnexpectedRPCDataFormatFromClient,
	86:  ErrorUnexpectedRPCDataFormatFromServer,
	87:  ErrorConnectionBrokenWithHint,
	90:  ErrorIntegrityError,
	91:  ErrorFileIntegrityError,
	92:  ErrorDecompressionError,
	95:  ErrorArchiveTAMInvalid,
	96:  ErrorArchiveTAMRequiredError,
	97:  ErrorTAMInvalid,
	98:  ErrorTAMRequiredError,
	99:  ErrorTAMUnsupportedSuiteError,
	100: ErrorFileChangedWarning,
	101: ErrorIncludePatternNeverMatchedWarning,
	102: ErrorBackupError,
	103: ErrorBackupRaceConditionError,
	104: ErrorBackupOSError,
	105: ErrorBackupPermissionError,
	106: ErrorBackupIOError,
	107: ErrorBackupFileNotFoundError,
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
	var exitCode = exitError.ExitCode()
	if nErr, ok := errorMap[exitCode]; ok {
		return errors.Join(WithExitError{ExitError: exitError}, nErr)
	}
	assert.Fail("exit code not handled")
	return err
}
