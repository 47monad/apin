package initropts

type PrometheusStore struct{}

type PrometheusBuilder struct {
	Opts []func(*PrometheusStore) error
}

func (p *PrometheusBuilder) Build() (*PrometheusStore, error) {
	return &PrometheusStore{}, nil
}

func Prometheus() *PrometheusBuilder {
	return &PrometheusBuilder{}
}
