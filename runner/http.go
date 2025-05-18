package runner

import (
	"net/http"
	"strconv"

	"github.com/47monad/zaal"
)

func (r *Runner) AddHttp(server *zaal.HTTPServerConfig, attacher func(*http.ServeMux)) {
	port := server.Port
	httpSrv := &http.Server{Addr: ":" + strconv.Itoa(port)}
	r.eg.Go(func() error {
		r.app.Logger().Info("starting http server", "port", port)
		m := http.NewServeMux()
		attacher(m)
		httpSrv.Handler = m
		return httpSrv.ListenAndServe()
	})
}
