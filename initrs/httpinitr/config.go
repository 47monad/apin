package httpinitr

// Config is the application-facing aggregate for named HTTP servers. Each
// server is constructed as a separate shell and tracked independently.
type Config struct {
	Servers map[string]ServerConfig `json:"servers" yaml:"servers"`
}

// ServerConfig configures one HTTP server. Timeout fields are optional seconds:
// an omitted field selects the documented default, while an explicit zero
// disables that timeout (useful for streaming). MaxHeaderBytes of zero selects
// http.DefaultMaxHeaderBytes.
type ServerConfig struct {
	Host              string `json:"host,omitempty" yaml:"host,omitempty" env:"http_host"`
	Port              int    `json:"port" yaml:"port" env:"http_port"`
	ReadHeaderTimeout *int   `json:"readHeaderTimeout,omitempty" yaml:"readHeaderTimeout,omitempty" env:"http_read_header_timeout"`
	ReadTimeout       *int   `json:"readTimeout,omitempty" yaml:"readTimeout,omitempty" env:"http_read_timeout"`
	WriteTimeout      *int   `json:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" env:"http_write_timeout"`
	IdleTimeout       *int   `json:"idleTimeout,omitempty" yaml:"idleTimeout,omitempty" env:"http_idle_timeout"`
	MaxHeaderBytes    int    `json:"maxHeaderBytes,omitempty" yaml:"maxHeaderBytes,omitempty" env:"http_max_header_bytes"`
}
