package client

import (
	"arco/backend/borg/types"
	"arco/backend/ent"
	"arco/backend/ssh"
	"fmt"
	"go.uber.org/zap"
)

type BorgClient struct {
	binaryPath string
	log        *zap.SugaredLogger
	db         *ent.Client
	channels   *types.Channels
}

func NewBorgClient(log *zap.SugaredLogger, dbClient *ent.Client, channels *types.Channels) *BorgClient {
	return &BorgClient{
		binaryPath: "bin/borg-linuxnewer64",
		log:        log,
		db:         dbClient,
		channels:   channels,
	}
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
		case notification := <-b.channels.Notification:
			notifications = append(notifications, notification)
		default:
			return notifications
		}
	}
}
