package httpinitr

// Config contains HTTP server configuration owned by httpinitr.
type Config struct {
	Port int `json:"port" yaml:"port" env:"http_port"`
}
