service: {
  name: "test"
  mode: "debug"
  mongodb: {
    username: ""
    hosts: ["127.0.0.1:27017"]
  }
  postgres: {}
  etcd: {
    timeout: 10
  }
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
