package client

import (
	"arco/backend/borg/types"
	"arco/backend/ent"
	"arco/backend/ssh"
	"context"
	"fmt"
	"go.uber.org/zap"
)

type BorgClient struct {
	ctx            context.Context
	binaryPath     string
	log            *zap.SugaredLogger
	db             *ent.Client
	inChan         *types.InputChannels
	outChan        *types.OutputChannels
	runningBackups []types.BackupIdentifier
	occupiedRepos  []int
}

type BI interface {
	HandleError(msg string, fErr *FrontendError)
}

func NewBorgClient(log *zap.SugaredLogger, dbClient *ent.Client, inChan *types.InputChannels, outChan *types.OutputChannels) *BorgClient {
	return &BorgClient{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
		db:         dbClient,
		inChan:     inChan,
		outChan:    outChan,
	}
}

func (b *BorgClient) Startup(ctx context.Context) {
	b.ctx = ctx
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

func (b *BorgClient) GetNotifications() []Notification {
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
		default:
			return notifications
		}
	}
}