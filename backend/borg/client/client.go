package client

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"arco/backend/ent"
	"arco/backend/ssh"
	"context"
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
)

const (
	DbusPath      = "/github/com/loomilabs/arco"
	DbusInterface = "github.com.loomilabs.arco"
)

type BorgClient struct {
	// Init
	log      *util.CmdLogger
	config   *Config
	db       *ent.Client
	inChan   *types.InputChannels
	outChan  *types.OutputChannels
	dbusConn *dbus.Conn

	// Startup
	ctx context.Context

	// State (runtime)
	runningBackups   []types.BackupIdentifier
	runningPruneJobs []types.BackupIdentifier
	occupiedRepos    []int
	startupErr       error
}

func NewBorgClient(
	log *zap.SugaredLogger,
	config *Config,
	dbClient *ent.Client,
	inChan *types.InputChannels,
	outChan *types.OutputChannels,
	dbusConn *dbus.Conn,
) *BorgClient {
	return &BorgClient{
		log:      util.NewCmdLogger(log),
		config:   config,
		db:       dbClient,
		inChan:   inChan,
		outChan:  outChan,
		dbusConn: dbusConn,
	}
}

// These clients separate the different types of operations that can be performed with the Borg client
// This makes it easier to expose them in a clean way to the frontend

// RepositoryClient is a client for repository related operations
type RepositoryClient BorgClient

// AppClient is a client for application related operations
type AppClient BorgClient

// BackupClient is a client for backup related operations
type BackupClient BorgClient

func (b *BorgClient) RepoClient() *RepositoryClient {
	return (*RepositoryClient)(b)
}

func (b *BorgClient) AppClient() *AppClient {
	return (*AppClient)(b)
}

func (b *BorgClient) BackupClient() *BackupClient {
	return (*BackupClient)(b)
}

func (b *BorgClient) Startup(ctx context.Context) {
	b.ctx = ctx

	if b.isTargetVersionInstalled(b.config.BorgVersion) {
		b.log.Info("Borg binary is installed")
	} else {
		b.log.Info("Installing Borg binary")
		if err := b.installBorgBinary(); err != nil {
			b.log.Error("Failed to install Borg binary: ", err)
			b.startupErr = err
		} else {
			// Check again to make sure the binary was installed correctly
			if !b.isTargetVersionInstalled(b.config.BorgVersion) {
				b.log.Error("Failed to install Borg binary: version mismatch")
				b.startupErr = fmt.Errorf("failed to install Borg binary: version mismatch")
			}
		}
	}
	b.registerDbusCalls()
}

func (b *BorgClient) registerDbusCalls() {
	err := b.dbusConn.Export(b, DbusPath, DbusInterface)
	if err != nil {
		b.log.Fatalf("Failed to export dbus interface: %s", err)
	}

	reply, err := b.dbusConn.RequestName(DbusInterface, dbus.NameFlagDoNotQueue)
	if err != nil {
		b.log.Fatalf("Failed to request dbus name: %s", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		b.log.Fatalf("DBus name is already owned by another process")
	}
}

func (b *BorgClient) Wakeup() *dbus.Error {
	b.log.Debug("Received wakeup command")
	runtime.WindowShow(b.ctx)
	return nil
}

func (b *BorgClient) isTargetVersionInstalled(targetVersion string) bool {
	// Check if the binary is installed
	if _, err := os.Stat(b.config.BorgPath); err == nil {
		version, err := b.Version()
		// Check if the version is correct
		return err == nil && version == targetVersion
	}
	return false
}

func (b *BorgClient) installBorgBinary() error {
	// Delete old binary if it exists
	if _, err := os.Stat(b.config.BorgPath); err == nil {
		if err := os.Remove(b.config.BorgPath); err != nil {
			return err
		}
	}

	file, err := b.config.Binaries.ReadFile(util.GetBorgBinaryPathX())
	if err != nil {
		return err
	}
	return os.WriteFile(b.config.BorgPath, file, 0755)
}

func (b *BorgClient) createSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	b.log.Info(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}

func (b *BorgClient) Version() (string, error) {
	cmd := exec.Command(b.config.BorgPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// Output is in the format "borg 1.2.8\n"
	// We want to return "1.2.8"
	return strings.TrimSpace(strings.TrimPrefix(string(out), "borg ")), nil
}

func (b *AppClient) GetStartupError() Notification {
	var message string
	if b.startupErr != nil {
		message = b.startupErr.Error()
	}
	return Notification{
		Message: message,
		Level:   LevelError,
	}
}

func (b *AppClient) HandleError(msg string, fErr *FrontendError) {
	errStr := ""
	if fErr != nil {
		if fErr.Message != "" && fErr.Stack != "" {
			errStr = fmt.Sprintf("%s\n%s", fErr.Message, fErr.Stack)
		} else if fErr.Message != "" {
			errStr = fErr.Message
		}
	}

	// We don't want to show the stack trace from the go code because the error comes from the frontend
	b.log.WithOptions(zap.AddCallerSkip(9999999)).
		Errorf(fmt.Sprintf("%s: %s", msg, errStr))
}

func (b *AppClient) GetNotifications() []Notification {
	notifications := make([]Notification, 0)
	for {
		select {
		case result := <-b.outChan.FinishBackup:
			if result.Err != nil {
				notifications = append(notifications, Notification{
					Message: fmt.Sprintf("Backup job failed after %s: %s", result.EndTime.Sub(result.StartTime), result.Err),
					Level:   LevelError,
				})
			} else {
				notifications = append(notifications, Notification{
					Message: fmt.Sprintf("Backup job completed in %s", result.EndTime.Sub(result.StartTime)),
					Level:   LevelInfo,
				})
			}

			//	Remove backup from runningBackups and occupiedRepos
			for i, id := range b.runningBackups {
				if id == result.Id {
					b.runningBackups = append(b.runningBackups[:i], b.runningBackups[i+1:]...)
					break
				}
			}
			for i, id := range b.occupiedRepos {
				if id == result.Id.RepositoryId {
					b.occupiedRepos = append(b.occupiedRepos[:i], b.occupiedRepos[i+1:]...)
					break
				}
			}
		case result := <-b.outChan.FinishPrune:
			if result.PruneErr != nil || result.CompactErr != nil {
				notifications = append(notifications, Notification{
					Message: fmt.Sprintf("Prune job failed after %s: %s", result.EndTime.Sub(result.StartTime), result.PruneErr),
					Level:   LevelError,
				})
			} else {
				notifications = append(notifications, Notification{
					Message: fmt.Sprintf("Prune job completed in %s", result.EndTime.Sub(result.StartTime)),
					Level:   LevelInfo,
				})
			}

			// Remove prune job from runningPruneJobs and occupiedRepos
			for i, id := range b.runningPruneJobs {
				if id == result.Id {
					b.runningPruneJobs = append(b.runningPruneJobs[:i], b.runningPruneJobs[i+1:]...)
					break
				}
			}
			for i, id := range b.occupiedRepos {
				if id == result.Id.RepositoryId {
					b.occupiedRepos = append(b.occupiedRepos[:i], b.occupiedRepos[i+1:]...)
					break
				}
			}
		default:
			return notifications
		}
	}
}
