package etcdinitr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestWithConfigAndOptionPrecedence(t *testing.T) {
	store := &Store{Opts: &clientv3.Config{}}
	configTimeout := 3
	require.NoError(t, WithConfig(&Config{
		Endpoints: "config-a:2379,config-b:2379",
		Username:  "config-user",
		Password:  "config-pass",
		Timeout:   &configTimeout,
	})(store))
	require.NoError(t, WithEndpoints([]string{"option:2379"})(store))
	require.NoError(t, WithUsername("option-user")(store))
	require.NoError(t, WithTimeout(5*time.Second)(store))

	assert.Equal(t, []string{"option:2379"}, store.Opts.Endpoints)
	assert.Equal(t, "option-user", store.Opts.Username)
	assert.Equal(t, "config-pass", store.Opts.Password)
	assert.Equal(t, 5*time.Second, store.Opts.DialTimeout)
}

func TestWithConfigRejectsNonPositiveTimeout(t *testing.T) {
	zero := 0
	_, err := New(t.Context(), WithConfig(&Config{Timeout: &zero}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestNewRejectsNegativeDialTimeout(t *testing.T) {
	_, err := New(t.Context(), WithEndpoints([]string{"127.0.0.1:2379"}), WithTimeout(-time.Second))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestDefaultClientConfig(t *testing.T) {
	store, err := newStore(nil)
	require.NoError(t, err)
	assert.Empty(t, store.Opts.Endpoints)
	assert.Zero(t, store.Opts.DialTimeout)
}

func TestNewFromOptionsExposesNativeClient(t *testing.T) {
	shell, err := New(t.Context(),
		WithEndpoints([]string{"127.0.0.1:2379"}),
		WithTimeout(time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, shell.Client)
	require.NoError(t, shell.Close(t.Context()))
}
