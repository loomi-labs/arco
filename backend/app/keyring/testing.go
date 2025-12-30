package keyring

import (
	"github.com/99designs/keyring"
	"go.uber.org/zap"
)

// NewTestService creates a keyring service with an in-memory backend for testing.
// This is exported so other packages can use it in their tests.
func NewTestService(log *zap.SugaredLogger) *Service {
	items := make(map[string]keyring.Item)
	ring := &arrayKeyring{items: items}

	return &Service{
		log:  log,
		ring: ring,
	}
}

// arrayKeyring is a simple in-memory keyring implementation for testing
type arrayKeyring struct {
	items map[string]keyring.Item
}

func (a *arrayKeyring) Get(key string) (keyring.Item, error) {
	item, ok := a.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (a *arrayKeyring) GetMetadata(_ string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (a *arrayKeyring) Set(item keyring.Item) error {
	a.items[item.Key] = item
	return nil
}

func (a *arrayKeyring) Remove(key string) error {
	delete(a.items, key)
	return nil
}

func (a *arrayKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(a.items))
	for k := range a.items {
		keys = append(keys, k)
	}
	return keys, nil
}
