package manifest

type LoggingConfig struct {
	Level string `json:"level" env:"log_level"`
}

type PostgresConfig struct {
	URI         string             `json:"uri" env:"postgres_uri"`
	Host        string             `json:"host" env:"postgres_host"`
	Port        int                `json:"port" env:"postgres_port"`
	Username    string             `json:"username" env:"postgres_username"`
	Password    string             `json:"password" env:"postgres_password"`
	DBName      string             `json:"dbName" env:"postgres_db_name"`
	SSLMode     string             `json:"sslMode,omitempty" env:"postgres_ssl_mode"`
	AppName     string             `json:"appName,omitempty" env:"postgres_app_name"`
	ConnTimeout int                `json:"connTimeout,omitempty" env:"postgres_conn_timeout"`
	Mode        string             `json:"mode" env:"postgres_mode"`
	Pool        PostgresPoolConfig `json:"pool,omitempty"`
}

type PostgresPoolConfig struct {
	MaxConns            int `json:"maxConns,omitempty" env:"postgres_pool_max_conns"`
	MinConns            int `json:"minConns,omitempty" env:"postgres_pool_min_conns"`
	MaxConnLifetime     int `json:"maxConnLifetime,omitempty" env:"postgres_pool_max_conn_lifetime"`
	MaxConnIdleTime     int `json:"maxConnIdleTime,omitempty" env:"postgres_pool_max_conn_idle_time"`
	HealthCheckInterval int `json:"healthCheckInterval,omitempty" env:"postgres_pool_health_check_interval"`
}

type MongodbOptions struct {
	ReplicaSet string `json:"replicaSet"`
}

type MongodbConfig struct {
	URI      string         `json:"uri" env:"mongodb_uri"`
	Username string         `json:"username" env:"mongodb_username"`
	Password string         `json:"password" env:"mongodb_password"`
	DBName   string         `json:"dbName" env:"mongodb_db_name" envDeprecated:"mongodb_dbname"`
	Hosts    []string       `json:"hosts"`
	Options  MongodbOptions `json:"options"`
}

type EtcdConfig struct {
	Endpoints string `json:"endpoints" env:"etcd_endpoints"`
	Username  string `json:"username" env:"etcd_username"`
	Password  string `json:"password" env:"etcd_password"`
	Timeout   int    `json:"timeout" env:"etcd_timeout"`
}

type RabbitMQConfig struct {
	URI              string `json:"uri" env:"rabbitmq_uri"`
	MinRetryInterval int    `json:"minRetryInterval"`
	MaxRetryInterval int    `json:"maxRetryInterval"`
}

type PrometheusConfig struct {
	GRPCMetrics bool `json:"grpcMetrics"`
}

type GRPCFeatures struct {
	Reflection  bool `json:"reflection"`
	HealthCheck bool `json:"healthCheck"`
	Logging     bool `json:"logging"`
}

type GRPCClientConfig struct {
	Address string `json:"address" env:"grpc_client_address"`
}

type GRPCServerConfig struct {
	Port     int          `json:"port" env:"grpc_port"`
	Features GRPCFeatures `json:"features"`
}

type GRPCConfig struct {
	Clients map[string]GRPCClientConfig `json:"clients"`
	Servers map[string]GRPCServerConfig `json:"servers"`
}

type HTTPServerConfig struct {
	Port int `json:"port" env:"http_port"`
}

type HTTPConfig struct {
	Servers map[string]HTTPServerConfig `json:"servers"`
}

type Config struct {
	Name       string            `json:"name"`
	Title      string            `json:"title"`
	Version    string            `json:"version"`
	Env        string            `json:"env" env:"env"`
	Mode       string            `json:"mode" env:"mode"`
	Host       string            `json:"host" env:"host"`
	Logging    LoggingConfig     `json:"logging"`
	Mongodb    *MongodbConfig    `json:"mongodb,omitempty"`
	Postgres   *PostgresConfig   `json:"postgres,omitempty"`
	Etcd       *EtcdConfig       `json:"etcd,omitempty"`
	RabbiMQ    *RabbitMQConfig   `json:"rabbitmq,omitempty"`
	Prometheus *PrometheusConfig `json:"prometheus,omitempty"`
	GRPC       *GRPCConfig       `json:"grpc,omitempty"`
	HTTP       *HTTPConfig       `json:"http,omitempty"`
}
