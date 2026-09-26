package manifest

type LoggingConfig struct {
	Level string `json:"level" env:"log_level"`
}

type HTTPServerConfig struct {
	Port int `json:"port" env:"http_port"`
}

type HTTPConfig struct {
	Servers map[string]HTTPServerConfig `json:"servers"`
}

type Config struct {
	Name    string        `json:"name"`
	Title   string        `json:"title"`
	Version string        `json:"version"`
	Env     string        `json:"env" env:"env"`
	Mode    string        `json:"mode" env:"mode"`
	Host    string        `json:"host" env:"host"`
	Logging LoggingConfig `json:"logging"`
	HTTP    *HTTPConfig   `json:"http,omitempty"`
}
