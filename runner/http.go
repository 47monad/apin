package runner

import (
	"net/http"
	"strconv"

	"github.com/47monad/apin/internal/logger"
)

func (r *Runner) AddHttp(attacher func(*http.ServeMux)) {
	port := r.app.GetConfig().HTTP.Port
	httpSrv := &http.Server{Addr: ":" + strconv.Itoa(port)}
	r.eg.Go(func() error {
		r.app.Logger().Info("starting http server", logger.LogFields{"port": port})
		m := http.NewServeMux()
		attacher(m)
		httpSrv.Handler = m
		return httpSrv.ListenAndServe()
	})
}
