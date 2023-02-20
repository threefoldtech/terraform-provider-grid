package client

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/threefoldtech/rmb-sdk-go/direct"
	"github.com/threefoldtech/terraform-provider-grid/pkg/subi"
)

type twinDB struct {
	cache *cache.Cache
	sub   subi.SubstrateExt
}

// NewTwinDB creates a new twinDBImpl instance, with a non expiring cache.
func NewTwinDB(sub subi.SubstrateExt) direct.TwinDB {
	return &twinDB{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
		sub:   sub,
	}
}

// GetTwin gets Twin from cache if present. if not, gets it from substrate client and caches it.
func (t *twinDB) Get(id uint32) (direct.Twin, error) {
	cachedValue, ok := t.cache.Get(fmt.Sprint(id))
	if ok {
		return cachedValue.(direct.Twin), nil
	}
	substrateTwin, err := t.sub.GetTwin(id)
	if err != nil {
		return direct.Twin{}, errors.Wrapf(err, "could net get twin with id %d", id)
	}

	var relay *string

	if substrateTwin.Relay.HasValue {
		relay = &substrateTwin.Relay.AsValue
	}

	_, PK := substrateTwin.Pk.Unwrap()
	twin := direct.Twin{
		ID:        id,
		PublicKey: substrateTwin.Account.PublicKey(),
		Relay:     relay,
		E2EKey:    PK,
	}

	err = t.cache.Add(fmt.Sprint(id), twin, cache.DefaultExpiration)
	if err != nil {
		return direct.Twin{}, errors.Wrapf(err, "could not set cache for twin with id %d", id)
	}

	return twin, nil
}

// GetByPk returns a twin's id using its public key
func (t *twinDB) GetByPk(pk []byte) (uint32, error) {
	return t.sub.GetTwinByPubKey(pk)
}
