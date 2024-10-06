package runner

import (
	"github.com/47monad/apin/internal/logger"
	"net/http"
	"strconv"
)

func (r *Runner) AddHttp(attacher func(*http.ServeMux)) {
	port := r.app.Config.Ports["http"]
	httpSrv := &http.Server{Addr: ":" + strconv.Itoa(int(port))}
	r.eg.Go(func() error {
		r.app.Logger.Info("starting http server", logger.LogFields{"port": port})
		m := http.NewServeMux()
		attacher(m)
		httpSrv.Handler = m
		return httpSrv.ListenAndServe()
	})
}
