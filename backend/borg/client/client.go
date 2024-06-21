package client

import (
	"arco/backend/borg/types"
	"arco/backend/borg/util"
	"arco/backend/borg/worker"
	"arco/backend/ent"
	"arco/backend/ssh"
	"context"
	"fmt"
	"github.com/getlantern/systray"
	"github.com/godbus/dbus/v5"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strings"
)

const (
	AppName       = "Arco"
	DbusPath      = "/github/com/loomilabs/arco"
	DbusInterface = "github.com.loomilabs.arco"
)

type BorgClient struct {
	// Init
	log      *util.CmdLogger
	config   *types.Config
	db       *ent.Client
	inChan   *types.InputChannels
	outChan  *types.OutputChannels
	dbusConn *dbus.Conn
	worker   *worker.Worker

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
	config *types.Config,
	dbClient *ent.Client,
	dbusConn *dbus.Conn,
) *BorgClient {
	inChan := types.NewInputChannels()
	outChan := types.NewOutputChannels()
	return &BorgClient{
		log:      util.NewCmdLogger(log),
		config:   config,
		db:       dbClient,
		inChan:   inChan,
		outChan:  outChan,
		dbusConn: dbusConn,
		worker:   worker.NewWorker(log, config.BorgPath, inChan, outChan),
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

	if err := b.registerDbusCalls(); err != nil {
		b.startupErr = err
		return
	}

	systray.Run(b.onSystrayReady, b.onSystrayExit)

	if err := b.ensureBorgBinary(); err != nil {
		b.startupErr = err
		return
	}

	go b.worker.Run()
}

func (b *BorgClient) Shutdown(_ context.Context) {
	b.log.Info(fmt.Sprintf("Shutting down %s", AppName))
	b.worker.Stop()
	err := b.db.Close()
	if err != nil {
		b.log.Error("Failed to close database connection")
	}
	os.Exit(0)
}

func (b *BorgClient) BeforeClose(ctx context.Context) (prevent bool) {
	b.log.Debug("Received beforeclose command")
	runtime.WindowHide(ctx)
	return true
}

func (b *BorgClient) Wakeup() *dbus.Error {
	b.log.Debug("Received wakeup command")
	runtime.WindowShow(b.ctx)
	return nil
}

func (b *BorgClient) registerDbusCalls() error {
	err := b.dbusConn.Export(b, DbusPath, DbusInterface)
	if err != nil {
		return fmt.Errorf("failed to export dbus interface: %w", err)
	}

	reply, err := b.dbusConn.RequestName(DbusInterface, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request dbus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("failed to request dbus name: name already taken")
	}
	return nil
}

func (b *BorgClient) ensureBorgBinary() error {
	if !b.isTargetVersionInstalled(b.config.BorgVersion) {
		b.log.Info("Installing Borg binary")
		if err := b.installBorgBinary(); err != nil {
			return fmt.Errorf("failed to install Borg binary: %w", err)
		} else {
			// Check again to make sure the binary was installed correctly
			if !b.isTargetVersionInstalled(b.config.BorgVersion) {
				return fmt.Errorf("failed to install Borg binary: version mismatch")
			}
		}
	}
	return nil
}

func (b *BorgClient) isTargetVersionInstalled(targetVersion string) bool {
	// Check if the binary is installed
	if _, err := os.Stat(b.config.BorgPath); err == nil {
		version, err := b.version()
		// Check if the version is correct
		return err == nil && version == targetVersion
	}
	return false
}

func (b *BorgClient) version() (string, error) {
	cmd := exec.Command(b.config.BorgPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// Output is in the format "borg 1.2.8\n"
	// We want to return "1.2.8"
	return strings.TrimSpace(strings.TrimPrefix(string(out), "borg ")), nil
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

func (b *BorgClient) onSystrayReady() {
	systray.SetIcon(b.config.Icon)
	systray.SetTitle(AppName)
	systray.SetTooltip(AppName)

	mOpen := systray.AddMenuItem(fmt.Sprintf("Open %s", AppName), fmt.Sprintf("Open %s", AppName))
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %s", AppName), fmt.Sprintf("Quit %s", AppName))

	// Sets the icon of a menu item. Only available on Mac and Windows.
	mOpen.SetIcon(b.config.Icon)
	mQuit.SetIcon(b.config.Icon)

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				runtime.WindowShow(b.ctx)
			case <-mQuit.ClickedCh:
				b.Shutdown(b.ctx)
			}
		}
	}()
}

func (b *BorgClient) onSystrayExit() {
	b.Shutdown(b.ctx)
}

// TODO: remove or move somewhere else
func (b *BorgClient) createSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	b.log.Info(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}
