package mongoinitr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithConfigAndOptionPrecedence(t *testing.T) {
	store, err := resolveStore(
		WithConfig(&Config{URI: "mongodb://config:27017", DBName: "config-db"}),
		WithURI("mongodb://option:27017"),
		WithDBName("option-db"),
	)
	require.NoError(t, err)
	assert.Equal(t, "mongodb://option:27017", store.Opts.GetURI())
	assert.Equal(t, "option-db", store.DBName)
}

func TestResolveStoreDefaultsAndValidation(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		store, err := resolveStore()
		require.NoError(t, err)
		assert.Equal(t, defaultPingTimeout, store.PingTimeout)
	})

	t.Run("invalid URI", func(t *testing.T) {
		_, err := resolveStore(WithConfig(&Config{URI: "not a mongodb uri"}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MongoDB")
	})

	t.Run("non-positive ping timeout", func(t *testing.T) {
		_, err := resolveStore(WithPingTimeout(0))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ping timeout")
	})
}
