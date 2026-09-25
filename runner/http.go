package runner

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/47monad/apin/manifest"
)

// AddHTTPServer registers an HTTP server on the configured port. The handler
// is assembled immediately from attacher, so the server is fully configured
// before Run. Stop drains its in-flight requests.
func (r *Runner) AddHTTPServer(server *manifest.HTTPServerConfig, attacher func(*http.ServeMux)) *Runner {
	if server == nil {
		r.logger.Error(nil, "AddHTTPServer: nil server config, skipping")
		return r
	}
	port := server.Port

	mux := http.NewServeMux()
	if attacher != nil {
		attacher(mux)
	}
	httpSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}
	r.trackHTTPServer(httpSrv)

	r.Add(func() error {
		r.logger.Info("starting http server", "port", port)
		err := httpSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			// Shutdown closed the server; that is a clean exit.
			return nil
		}
		return err
	})
	return r
}
