service: {
  name: "test"
  mode: "debug"
  rabbitmq: {}
  // logging: {
  //   level: "error"
  // }
  http: {
    servers: main: {
      port: 8787
    }
  }
  grpc: {
    clients: {
      uwc: {
        address: "default.com/here"
      }
    }
    servers: {
      main: {
        port: 9567
        features: {
          healthCheck: true
        }
      }
    }
  }
}
