package borg

import (
	"arco/backend/ent"
	"arco/backend/ssh"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
)

type BorgClient struct {
	binaryPath          string
	log                 *zap.SugaredLogger
	client              *ent.Client
	shutdownChannel     chan struct{}
	startBackupChannel  chan backupJob
	finishBackupChannel chan finishBackupJob
	notificationChannel chan string
}

func NewBorgClient(log *zap.SugaredLogger, client *ent.Client) *BorgClient {
	return &BorgClient{
		binaryPath:          "bin/borg-linuxnewer64",
		log:                 log,
		client:              client,
		shutdownChannel:     make(chan struct{}),
		startBackupChannel:  make(chan backupJob),
		finishBackupChannel: make(chan finishBackupJob),
		notificationChannel: make(chan string),
	}
}

func (b *BorgClient) StartDaemon() {
	b.log.Info("Starting BorgClient daemon")

	// Start a goroutine that runs all background tasks
	go func() {
		for {
			select {
			case job := <-b.startBackupChannel:
				b.log.Info("Starting backup job")
				go runBackup(job, b.finishBackupChannel)
			case result := <-b.finishBackupChannel:
				duration := result.endTime.Sub(result.startTime)
				if result.err != nil {
					b.log.Error(fmt.Sprintf("Backup job failed after %s: %s", duration, result.err))
				} else {
					b.log.Info(fmt.Sprintf("Backup job completed in %s", duration))
				}
				b.log.Debug(fmt.Sprintf("Command: %s", result.cmd))
				b.notificationChannel <- fmt.Sprintf("Backup job completed in %s", duration)
			case <-b.shutdownChannel:
				b.log.Debug("Shutting down background tasks")
				return
			}
		}
	}()
}

func (b *BorgClient) StopDaemon() {
	b.log.Info("Stopping BorgClient daemon")
	close(b.shutdownChannel)
}

func createEnv(password string) []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", password),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func getEnv() []string {
	sshOptions := []string{
		"-oBatchMode=yes",
		"-oStrictHostKeyChecking=accept-new",
		"-i ~/sshtest/id_storage_test",
	}
	env := append(
		os.Environ(),
		fmt.Sprintf("BORG_RSH=%s", fmt.Sprintf("ssh %s", strings.Join(sshOptions, " "))),
	)
	return env
}

func getTestEnvOverride() []string {
	passphrase := os.Getenv("BORG_PASSPHRASE")
	env := append(
		getEnv(),
		fmt.Sprintf("BORG_PASSPHRASE=%s", passphrase),
		fmt.Sprintf("BORG_NEW_PASSPHRASE=%s", passphrase),
	)
	return env
}

func (b *BorgClient) createSSHKeyPair() (string, error) {
	pair, err := ssh.GenerateKeyPair()
	if err != nil {
		return "", err
	}
	b.log.Info(fmt.Sprintf("Generated SSH key pair: %s", pair.AuthorizedKey()))
	return pair.AuthorizedKey(), nil
}

func (b *BorgClient) HandleError(msg string, fErr *FrontendError) {
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

func (b *BorgClient) GetNotifications() []string {
	notifications := make([]string, 0)
	for {
		select {
		case notification := <-b.notificationChannel:
			notifications = append(notifications, notification)
		default:
			return notifications
		}
	}
}
