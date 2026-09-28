package grpcinitr

// Config is the application-facing aggregate for named gRPC resources.
// Each resource is constructed as a separate shell and can be tracked
// independently by the application's lifecycle owner.
type Config struct {
	Servers map[string]ServerConfig `json:"servers" yaml:"servers"`
	Clients map[string]ClientConfig `json:"clients" yaml:"clients"`
}

// ServerConfig configures one gRPC server.
type ServerConfig struct {
	Port        int  `json:"port" yaml:"port" env:"grpc_port"`
	Reflection  bool `json:"reflection" yaml:"reflection" env:"grpc_reflection"`
	HealthCheck bool `json:"healthCheck" yaml:"healthCheck" env:"grpc_health_check"`
}

// ClientConfig configures one outgoing gRPC connection.
type ClientConfig struct {
	Target string `json:"target" yaml:"target" env:"grpc_target"`
}
