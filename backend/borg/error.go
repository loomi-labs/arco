package borg

import (
	"errors"
	"fmt"
	gocmd "github.com/go-cmd/cmd"
	"os/exec"
)

// ErrorCategory represents the category of a Borg error
type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
	CategoryGeneral
	CategoryRepository
	CategoryArchive
	CategoryKey
	CategoryPassphrase
	CategoryCache
	CategoryLock
	CategoryConnection
	CategoryIntegrity
	CategoryBackup
	CategoryPermission
	CategoryRuntime
)

// String returns the string representation of an ErrorCategory
func (c ErrorCategory) String() string {
	switch c {
	case CategoryGeneral:
		return "general"
	case CategoryRepository:
		return "repository"
	case CategoryArchive:
		return "archive"
	case CategoryKey:
		return "key"
	case CategoryPassphrase:
		return "passphrase"
	case CategoryCache:
		return "cache"
	case CategoryLock:
		return "lock"
	case CategoryConnection:
		return "connection"
	case CategoryIntegrity:
		return "integrity"
	case CategoryBackup:
		return "backup"
	case CategoryPermission:
		return "permission"
	case CategoryRuntime:
		return "runtime"
	default:
		return "unknown"
	}
}

// BorgError represents any error that can occur during a Borg operation
type BorgError struct {
	// Core error information
	ExitCode int    // Borg exit code (0 means no error from Borg)
	Message  string // Human-readable error message

	// Underlying error (could be execution error, network error, etc.)
	Underlying error

	// Categorization
	Category ErrorCategory
}

// Error implements the error interface
func (e *BorgError) Error() string {
	return e.Message
}

// Is implements error matching for errors.Is
func (e *BorgError) Is(target error) bool {
	if borgErr, ok := target.(*BorgError); ok {
		return e.ExitCode == borgErr.ExitCode
	}
	return false
}

// Unwrap returns the underlying error for errors.Unwrap
func (e *BorgError) Unwrap() error {
	return e.Underlying
}

// IsLockError returns true if this is a lock-related error
func (e *BorgError) IsLockError() bool {
	return e.Category == CategoryLock
}

// BorgWarning represents a warning that can occur during a Borg operation
type BorgWarning struct {
	// Core warning information
	ExitCode int    // Borg exit code (1, 100-107)
	Message  string // Human-readable warning message

	// Underlying error if any
	Underlying error

	// Categorization
	Category ErrorCategory
}

// Error implements the error interface
func (w *BorgWarning) Error() string {
	return w.Message
}

// Is implements error matching for errors.Is
func (w *BorgWarning) Is(target error) bool {
	if borgWarning, ok := target.(*BorgWarning); ok {
		return w.ExitCode == borgWarning.ExitCode
	}
	return false
}

// Unwrap returns the underlying error for errors.Unwrap
func (w *BorgWarning) Unwrap() error {
	return w.Underlying
}

/***********************************/
/********** Borg Errors ************/
/***********************************/

// Predefined BorgError instances for each exit code
var (
	ErrDefault                                 = &BorgError{ExitCode: 2, Message: "error", Category: CategoryGeneral}
	ErrorCancelledByUser                       = &BorgError{ExitCode: 3, Message: "cancelled by user", Category: CategoryGeneral}
	ErrorCommandError                          = &BorgError{ExitCode: 4, Message: "command error", Category: CategoryGeneral}
	ErrorPlaceholderError                      = &BorgError{ExitCode: 5, Message: "placeholder error", Category: CategoryGeneral}
	ErrorInvalidPlaceholder                    = &BorgError{ExitCode: 6, Message: "invalid placeholder", Category: CategoryGeneral}
	ErrorRepositoryAlreadyExists               = &BorgError{ExitCode: 10, Message: "repository already exists", Category: CategoryRepository}
	ErrorRepositoryAtticRepository             = &BorgError{ExitCode: 11, Message: "attic repository detected", Category: CategoryRepository}
	ErrorRepositoryCheckNeeded                 = &BorgError{ExitCode: 12, Message: "repository check needed", Category: CategoryRepository}
	ErrorRepositoryDoesNotExist                = &BorgError{ExitCode: 13, Message: "repository does not exist", Category: CategoryRepository}
	ErrorRepositoryInsufficientFreeSpaceError  = &BorgError{ExitCode: 14, Message: "insufficient free space", Category: CategoryRepository}
	ErrorRepositoryInvalidRepository           = &BorgError{ExitCode: 15, Message: "invalid repository", Category: CategoryRepository}
	ErrorRepositoryInvalidRepositoryConfig     = &BorgError{ExitCode: 16, Message: "invalid repository config", Category: CategoryRepository}
	ErrorRepositoryObjectNotFound              = &BorgError{ExitCode: 17, Message: "object not found in repository", Category: CategoryRepository}
	ErrorRepositoryParentPathDoesNotExist      = &BorgError{ExitCode: 18, Message: "parent path does not exist", Category: CategoryRepository}
	ErrorRepositoryPathAlreadyExists           = &BorgError{ExitCode: 19, Message: "path already exists", Category: CategoryRepository}
	ErrorRepositoryStorageQuotaExceeded        = &BorgError{ExitCode: 20, Message: "storage quota exceeded", Category: CategoryRepository}
	ErrorRepositoryPathPermissionDenied        = &BorgError{ExitCode: 21, Message: "permission denied to path", Category: CategoryPermission}
	ErrorMandatoryFeatureUnsupported           = &BorgError{ExitCode: 25, Message: "unsupported repository feature", Category: CategoryRepository}
	ErrorNoManifestError                       = &BorgError{ExitCode: 26, Message: "repository has no manifest", Category: CategoryRepository}
	ErrorUnsupportedManifestError              = &BorgError{ExitCode: 27, Message: "unsupported manifest envelope", Category: CategoryRepository}
	ErrorArchiveAlreadyExists                  = &BorgError{ExitCode: 30, Message: "archive already exists", Category: CategoryArchive}
	ErrorArchiveDoesNotExist                   = &BorgError{ExitCode: 31, Message: "archive does not exist", Category: CategoryArchive}
	ErrorArchiveIncompatibleFilesystemEncoding = &BorgError{ExitCode: 32, Message: "failed to encode filename", Category: CategoryArchive}
	ErrorKeyfileInvalidError                   = &BorgError{ExitCode: 40, Message: "invalid key data", Category: CategoryKey}
	ErrorKeyfileMismatchError                  = &BorgError{ExitCode: 41, Message: "mismatch between repository and key file", Category: CategoryKey}
	ErrorKeyfileNotFoundError                  = &BorgError{ExitCode: 42, Message: "no key file found", Category: CategoryKey}
	ErrorNotABorgKeyFile                       = &BorgError{ExitCode: 43, Message: "not a borg key backup", Category: CategoryKey}
	ErrorRepoKeyNotFoundError                  = &BorgError{ExitCode: 44, Message: "no key entry found", Category: CategoryKey}
	ErrorRepoIdMismatch                        = &BorgError{ExitCode: 45, Message: "key backup for different repository", Category: CategoryKey}
	ErrorUnencryptedRepo                       = &BorgError{ExitCode: 46, Message: "key management not available", Category: CategoryKey}
	ErrorUnknownKeyType                        = &BorgError{ExitCode: 47, Message: "unknown key type", Category: CategoryKey}
	ErrorUnsupportedPayloadError               = &BorgError{ExitCode: 48, Message: "unsupported payload type", Category: CategoryKey}
	ErrorNoPassphraseFailure                   = &BorgError{ExitCode: 50, Message: "cannot acquire a passphrase", Category: CategoryPassphrase}
	ErrorPasscommandFailure                    = &BorgError{ExitCode: 51, Message: "passcommand failed", Category: CategoryPassphrase}
	ErrorPassphraseWrong                       = &BorgError{ExitCode: 52, Message: "incorrect passphrase", Category: CategoryPassphrase}
	ErrorPasswordRetriesExceeded               = &BorgError{ExitCode: 53, Message: "exceeded password retries", Category: CategoryPassphrase}
	ErrorCacheInitAborted                      = &BorgError{ExitCode: 60, Message: "cache initialization aborted", Category: CategoryCache}
	ErrorCacheEncryptionMethodMismatch         = &BorgError{ExitCode: 61, Message: "encryption method mismatch", Category: CategoryCache}
	ErrorCacheRepositoryAccessAborted          = &BorgError{ExitCode: 62, Message: "repository access aborted", Category: CategoryCache}
	ErrorCacheRepositoryIDNotUnique            = &BorgError{ExitCode: 63, Message: "repository ID not unique", Category: CategoryCache}
	ErrorCacheRepositoryReplay                 = &BorgError{ExitCode: 64, Message: "cache newer than repository", Category: CategoryCache}
	ErrorLockError                             = &BorgError{ExitCode: 70, Message: "failed to acquire lock", Category: CategoryLock}
	ErrorLockErrorT                            = &BorgError{ExitCode: 71, Message: "failed to acquire lock with traceback", Category: CategoryLock}
	ErrorLockFailed                            = &BorgError{ExitCode: 72, Message: "failed to create/acquire lock", Category: CategoryLock}
	ErrorLockTimeout                           = &BorgError{ExitCode: 73, Message: "lock timeout", Category: CategoryLock}
	ErrorNotLocked                             = &BorgError{ExitCode: 74, Message: "failed to release lock (not locked)", Category: CategoryLock}
	ErrorNotMyLock                             = &BorgError{ExitCode: 75, Message: "failed to release lock (not by me)", Category: CategoryLock}
	ErrorConnectionClosed                      = &BorgError{ExitCode: 80, Message: "connection closed by remote host", Category: CategoryConnection}
	ErrorConnectionClosedWithHint              = &BorgError{ExitCode: 81, Message: "connection closed by remote host with hint", Category: CategoryConnection}
	ErrorInvalidRPCMethod                      = &BorgError{ExitCode: 82, Message: "invalid RPC method", Category: CategoryConnection}
	ErrorPathNotAllowed                        = &BorgError{ExitCode: 83, Message: "repository path not allowed", Category: CategoryPermission}
	ErrorRemoteRepositoryRPCServerOutdated     = &BorgError{ExitCode: 84, Message: "borg server too old", Category: CategoryConnection}
	ErrorUnexpectedRPCDataFormatFromClient     = &BorgError{ExitCode: 85, Message: "unexpected RPC data format from client", Category: CategoryConnection}
	ErrorUnexpectedRPCDataFormatFromServer     = &BorgError{ExitCode: 86, Message: "unexpected RPC data format from server", Category: CategoryConnection}
	ErrorConnectionBrokenWithHint              = &BorgError{ExitCode: 87, Message: "connection to remote host broken", Category: CategoryConnection}
	ErrorIntegrityError                        = &BorgError{ExitCode: 90, Message: "data integrity error", Category: CategoryIntegrity}
	ErrorFileIntegrityError                    = &BorgError{ExitCode: 91, Message: "file integrity check failed", Category: CategoryIntegrity}
	ErrorDecompressionError                    = &BorgError{ExitCode: 92, Message: "decompression error", Category: CategoryIntegrity}
	ErrorArchiveTAMInvalid                     = &BorgError{ExitCode: 95, Message: "archive TAM invalid", Category: CategoryIntegrity}
	ErrorArchiveTAMRequiredError               = &BorgError{ExitCode: 96, Message: "archive unauthenticated", Category: CategoryIntegrity}
	ErrorTAMInvalid                            = &BorgError{ExitCode: 97, Message: "TAM invalid", Category: CategoryIntegrity}
	ErrorTAMRequiredError                      = &BorgError{ExitCode: 98, Message: "manifest unauthenticated", Category: CategoryIntegrity}
	ErrorTAMUnsupportedSuiteError              = &BorgError{ExitCode: 99, Message: "unsupported suite", Category: CategoryIntegrity}
)

/***********************************/
/********** Borg Result ************/
/***********************************/

// BorgResult represents the result of a Borg operation
type BorgResult struct {
	Error           *BorgError   // nil if no error occurred
	Warning         *BorgWarning // nil if no warning occurred
	HasBeenCanceled bool
}

// IsCompletedWithSuccess returns true if there's no error and it has not been canceled
func (r *BorgResult) IsCompletedWithSuccess() bool {
	return r.Error == nil && !r.HasBeenCanceled
}

// HasError returns true if there's an error
func (r *BorgResult) HasError() bool {
	return r.Error != nil
}

// HasWarning returns true if there's a warning
func (r *BorgResult) HasWarning() bool {
	return r.Warning != nil
}

// GetError returns the error message if there's an error, empty string otherwise
func (r *BorgResult) GetError() string {
	if r.Error != nil {
		return r.Error.Error()
	}
	return ""
}

// GetWarning returns the warning message if there's a warning, empty string otherwise
func (r *BorgResult) GetWarning() string {
	if r.Warning != nil {
		return r.Warning.Error()
	}
	return ""
}

/***********************************/
/********* Borg Warnings ***********/
/***********************************/

// Predefined BorgWarning instances for each warning exit code
var (
	// Generic warnings (exit code 1)
	WarningGeneric = &BorgWarning{ExitCode: 1, Message: "warning", Category: CategoryGeneral}
	WarningBackup  = &BorgWarning{ExitCode: 1, Message: "backup warning", Category: CategoryBackup}

	// Specific warnings (exit codes 100-107)
	WarningFileChanged                = &BorgWarning{ExitCode: 100, Message: "file changed during backup", Category: CategoryBackup}
	WarningIncludePatternNeverMatched = &BorgWarning{ExitCode: 101, Message: "include pattern never matched", Category: CategoryBackup}
	WarningBackupError                = &BorgWarning{ExitCode: 102, Message: "backup error", Category: CategoryBackup}
	WarningBackupRaceCondition        = &BorgWarning{ExitCode: 103, Message: "file type or inode changed during backup", Category: CategoryBackup}
	WarningBackupOS                   = &BorgWarning{ExitCode: 104, Message: "backup OS error", Category: CategoryBackup}
	WarningBackupPermission           = &BorgWarning{ExitCode: 105, Message: "backup permission error", Category: CategoryBackup}
	WarningBackupIO                   = &BorgWarning{ExitCode: 106, Message: "backup IO error", Category: CategoryBackup}
	WarningBackupFileNotFound         = &BorgWarning{ExitCode: 107, Message: "backup file not found", Category: CategoryBackup}
)

// allBorgErrors contains all predefined BorgError instances for lookup
var allBorgErrors = []*BorgError{
	ErrDefault,
	ErrorCancelledByUser,
	ErrorCommandError,
	ErrorPlaceholderError,
	ErrorInvalidPlaceholder,
	ErrorRepositoryAlreadyExists,
	ErrorRepositoryAtticRepository,
	ErrorRepositoryCheckNeeded,
	ErrorRepositoryDoesNotExist,
	ErrorRepositoryInsufficientFreeSpaceError,
	ErrorRepositoryInvalidRepository,
	ErrorRepositoryInvalidRepositoryConfig,
	ErrorRepositoryObjectNotFound,
	ErrorRepositoryParentPathDoesNotExist,
	ErrorRepositoryPathAlreadyExists,
	ErrorRepositoryStorageQuotaExceeded,
	ErrorRepositoryPathPermissionDenied,
	ErrorMandatoryFeatureUnsupported,
	ErrorNoManifestError,
	ErrorUnsupportedManifestError,
	ErrorArchiveAlreadyExists,
	ErrorArchiveDoesNotExist,
	ErrorArchiveIncompatibleFilesystemEncoding,
	ErrorKeyfileInvalidError,
	ErrorKeyfileMismatchError,
	ErrorKeyfileNotFoundError,
	ErrorNotABorgKeyFile,
	ErrorRepoKeyNotFoundError,
	ErrorRepoIdMismatch,
	ErrorUnencryptedRepo,
	ErrorUnknownKeyType,
	ErrorUnsupportedPayloadError,
	ErrorNoPassphraseFailure,
	ErrorPasscommandFailure,
	ErrorPassphraseWrong,
	ErrorPasswordRetriesExceeded,
	ErrorCacheInitAborted,
	ErrorCacheEncryptionMethodMismatch,
	ErrorCacheRepositoryAccessAborted,
	ErrorCacheRepositoryIDNotUnique,
	ErrorCacheRepositoryReplay,
	ErrorLockError,
	ErrorLockErrorT,
	ErrorLockFailed,
	ErrorLockTimeout,
	ErrorNotLocked,
	ErrorNotMyLock,
	ErrorConnectionClosed,
	ErrorConnectionClosedWithHint,
	ErrorInvalidRPCMethod,
	ErrorPathNotAllowed,
	ErrorRemoteRepositoryRPCServerOutdated,
	ErrorUnexpectedRPCDataFormatFromClient,
	ErrorUnexpectedRPCDataFormatFromServer,
	ErrorConnectionBrokenWithHint,
	ErrorIntegrityError,
	ErrorFileIntegrityError,
	ErrorDecompressionError,
	ErrorArchiveTAMInvalid,
	ErrorArchiveTAMRequiredError,
	ErrorTAMInvalid,
	ErrorTAMRequiredError,
	ErrorTAMUnsupportedSuiteError,
}

// allBorgWarnings contains all predefined BorgWarning instances for lookup
var allBorgWarnings = []*BorgWarning{
	WarningGeneric,
	WarningBackup,
	WarningFileChanged,
	WarningIncludePatternNeverMatched,
	WarningBackupError,
	WarningBackupRaceCondition,
	WarningBackupOS,
	WarningBackupPermission,
	WarningBackupIO,
	WarningBackupFileNotFound,
}

func createRuntimeError(err error) *BorgError {
	return &BorgError{
		ExitCode:   -1,
		Message:    err.Error(),
		Underlying: err,
		Category:   CategoryRuntime,
	}
}

func toBorgResult(exitCode int) *BorgResult {
	if exitCode == 0 {
		return &BorgResult{}
	}
	if exitCode == 143 {
		return &BorgResult{
			HasBeenCanceled: true,
		}
	}

	for _, warning := range allBorgWarnings {
		if warning.ExitCode == exitCode {
			return &BorgResult{Warning: warning}
		}
	}

	for _, err := range allBorgErrors {
		if err.ExitCode == exitCode {
			return &BorgResult{Error: err}
		}
	}

	return &BorgResult{
		Error: &BorgError{
			ExitCode: exitCode,
			Message:  fmt.Sprintf("unknown borg error with exit code %d", exitCode),
			Category: CategoryUnknown,
		},
	}
}

// combinedOutputToBorgResult converts command output and error to a BorgResult
func combinedOutputToBorgResult(out []byte, err error) *BorgResult {
	if err == nil {
		return toBorgResult(0)
	}

	// Return the error if it is not an ExitError
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		// Include command output in the error message
		if len(out) > 0 {
			return &BorgResult{
				Error: createRuntimeError(fmt.Errorf("%s: %s", string(out), err)),
			}
		}
		return &BorgResult{
			Error: createRuntimeError(err),
		}
	}

	return toBorgResult(exitError.ExitCode())
}

// statusToBorgResult converts go-cmd status to a BorgResult
func statusToBorgResult(status gocmd.Status) *BorgResult {
	if status.Error != nil && status.Exit == 0 {
		// Execution error (command didn't run)
		return &BorgResult{
			Error: createRuntimeError(status.Error),
		}
	}

	return toBorgResult(status.Exit)
}
