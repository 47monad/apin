package httpinitr_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

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
