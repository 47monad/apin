package httpinitr

// Config is the application-facing aggregate for named HTTP servers. Each
// server is constructed as a separate shell and tracked independently.
type Config struct {
	Servers map[string]ServerConfig `json:"servers" yaml:"servers"`
}

// ServerConfig configures one HTTP server.
type ServerConfig struct {
	Port int `json:"port" yaml:"port" env:"http_port"`
}
