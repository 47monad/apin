package manifest

type HTTPServerConfig struct {
	Port int `json:"port" env:"http_port"`
}

type HTTPConfig struct {
	Servers map[string]HTTPServerConfig `json:"servers"`
}

type Config struct {
	Name    string      `json:"name"`
	Title   string      `json:"title"`
	Version string      `json:"version"`
	Env     string      `json:"env" env:"env"`
	Mode    string      `json:"mode" env:"mode"`
	Host    string      `json:"host" env:"host"`
	HTTP    *HTTPConfig `json:"http,omitempty"`
}
