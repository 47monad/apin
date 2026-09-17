service: {
  name: "test"
  postgres: {
    host: "localhost"
    port: 65535
    dbName: "testdb"
    sslMode: "require"
    connTimeout: 5
    mode: "single"
    pool: {
      maxConns: 10
      minConns: 2
      maxConnLifetime: 300
      maxConnIdleTime: 60
      healthCheckInterval: 30
    }
  }
}
