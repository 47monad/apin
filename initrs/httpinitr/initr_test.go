package httpinitr_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/httpinitr"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultsAndConfigOptionPrecedence(t *testing.T) {
	defaultShell, err := httpinitr.New(context.Background())
	require.NoError(t, err)
	require.Equal(t, ":4747", defaultShell.Server.Addr)
	require.NoError(t, defaultShell.Close(context.Background()))

	config := &httpinitr.ServerConfig{Port: 9000}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithConfig(config),
		httpinitr.WithPort(8123),
		httpinitr.WithHandler(handler),
	)
	require.NoError(t, err)
	require.Equal(t, ":8123", shell.Server.Addr)
	require.NoError(t, shell.Close(context.Background()))

	response := httptest.NewRecorder()
	shell.Server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestNamedConfigCreatesIndependentServerShells(t *testing.T) {
	config := httpinitr.Config{Servers: map[string]httpinitr.ServerConfig{
		"public": {Port: 8081},
		"admin":  {Port: 8082},
	}}
	public, err := httpinitr.NewServer(context.Background(), config.Servers["public"])
	require.NoError(t, err)
	admin, err := httpinitr.NewServer(context.Background(), config.Servers["admin"])
	require.NoError(t, err)
	defer func() { _ = public.Close(context.Background()) }()
	defer func() { _ = admin.Close(context.Background()) }()

	require.NotSame(t, public, admin)
	require.Equal(t, ":8081", public.Server.Addr)
	require.Equal(t, ":8082", admin.Server.Addr)
}

func TestNewRejectsInvalidPort(t *testing.T) {
	_, err := httpinitr.New(context.Background(), httpinitr.WithPort(65536))
	require.ErrorContains(t, err, "httpinitr: port must be between 1 and 65535")
}

func TestFinalPortValidationAllowsLaterOverride(t *testing.T) {
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithConfig(&httpinitr.ServerConfig{Port: 65536}),
		httpinitr.WithPort(8123),
	)
	require.NoError(t, err)
	require.Equal(t, ":8123", shell.Server.Addr)
	require.NoError(t, shell.Close(context.Background()))
}

func TestCloseClosesListenerBeforeServeStarts(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := probe.Addr().(*net.TCPAddr).Port
	require.NoError(t, probe.Close())

	shell, err := httpinitr.New(context.Background(), httpinitr.WithPort(port))
	require.NoError(t, err)
	listener, err := shell.Listen()
	require.NoError(t, err)
	require.NoError(t, shell.Close(context.Background()))
	_, err = listener.Accept()
	require.Error(t, err)
}

func TestServeAndClose(t *testing.T) {
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})),
	)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- shell.Serve(ctx, listener) }()

	response, err := http.Get("http://" + listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusNoContent, response.StatusCode)

	cancel()
	require.NoError(t, <-serveDone)
	require.NoError(t, shell.Close(context.Background()))
}

func TestServerDefaults(t *testing.T) {
	shell, err := httpinitr.New(context.Background())
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	server := shell.Server
	require.Equal(t, ":4747", server.Addr)
	require.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	require.Zero(t, server.ReadTimeout)
	require.Zero(t, server.WriteTimeout)
	require.Equal(t, 120*time.Second, server.IdleTimeout)
	require.Equal(t, http.DefaultMaxHeaderBytes, server.MaxHeaderBytes)
}

func TestServerAddressCompositionAndOverrides(t *testing.T) {
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithHost("127.0.0.1"),
		httpinitr.WithPort(9000),
		httpinitr.WithReadHeaderTimeout(3*time.Second),
		httpinitr.WithReadTimeout(4*time.Second),
		httpinitr.WithWriteTimeout(5*time.Second),
		httpinitr.WithIdleTimeout(6*time.Second),
		httpinitr.WithMaxHeaderBytes(4096),
	)
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	server := shell.Server
	require.Equal(t, "127.0.0.1:9000", server.Addr)
	require.Equal(t, 3*time.Second, server.ReadHeaderTimeout)
	require.Equal(t, 4*time.Second, server.ReadTimeout)
	require.Equal(t, 5*time.Second, server.WriteTimeout)
	require.Equal(t, 6*time.Second, server.IdleTimeout)
	require.Equal(t, 4096, server.MaxHeaderBytes)
}

func TestServerConfigMapsHardeningFields(t *testing.T) {
	readHeader, read, write, idle := 1, 2, 3, 4
	shell, err := httpinitr.NewServer(context.Background(), httpinitr.ServerConfig{
		Host:              "127.0.0.1",
		Port:              8123,
		ReadHeaderTimeout: &readHeader,
		ReadTimeout:       &read,
		WriteTimeout:      &write,
		IdleTimeout:       &idle,
		MaxHeaderBytes:    2048,
	})
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	server := shell.Server
	require.Equal(t, "127.0.0.1:8123", server.Addr)
	require.Equal(t, time.Second, server.ReadHeaderTimeout)
	require.Equal(t, 2*time.Second, server.ReadTimeout)
	require.Equal(t, 3*time.Second, server.WriteTimeout)
	require.Equal(t, 4*time.Second, server.IdleTimeout)
	require.Equal(t, 2048, server.MaxHeaderBytes)
}

// Explicit zero timeouts keep streaming requests and responses unbounded, and
// disable the header/idle defaults.
func TestStreamingZeroTimeouts(t *testing.T) {
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithReadHeaderTimeout(0),
		httpinitr.WithIdleTimeout(0),
		httpinitr.WithReadTimeout(0),
		httpinitr.WithWriteTimeout(0),
	)
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	server := shell.Server
	require.Zero(t, server.ReadHeaderTimeout)
	require.Zero(t, server.IdleTimeout)
	require.Zero(t, server.ReadTimeout)
	require.Zero(t, server.WriteTimeout)
}

func TestWithBaseContext(t *testing.T) {
	type contextKey struct{}
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithBaseContext(func(net.Listener) context.Context {
			return context.WithValue(context.Background(), contextKey{}, "base")
		}),
	)
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	require.NotNil(t, shell.Server.BaseContext)
	require.Equal(t, "base", shell.Server.BaseContext(nil).Value(contextKey{}))
}

func TestWithNativeServer(t *testing.T) {
	called := false
	shell, err := httpinitr.New(context.Background(),
		httpinitr.WithNativeServer(func(server *http.Server) error {
			called = true
			server.MaxHeaderBytes = 1234
			return nil
		}),
	)
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	require.True(t, called)
	require.Equal(t, 1234, shell.Server.MaxHeaderBytes)

	_, err = httpinitr.New(context.Background(),
		httpinitr.WithNativeServer(func(*http.Server) error { return errors.New("boom") }),
	)
	require.ErrorContains(t, err, "boom")
}

func TestNegativeServerSettingsRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  httpinitr.Option
		want string
	}{
		{name: "read header", opt: httpinitr.WithReadHeaderTimeout(-time.Second), want: "readHeaderTimeout"},
		{name: "read", opt: httpinitr.WithReadTimeout(-time.Second), want: "readTimeout"},
		{name: "write", opt: httpinitr.WithWriteTimeout(-time.Second), want: "writeTimeout"},
		{name: "idle", opt: httpinitr.WithIdleTimeout(-time.Second), want: "idleTimeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := httpinitr.New(context.Background(), tc.opt)
			require.ErrorContains(t, err, tc.want)
		})
	}

	_, err := httpinitr.New(context.Background(), httpinitr.WithMaxHeaderBytes(-1))
	require.ErrorContains(t, err, "maxHeaderBytes")
}
