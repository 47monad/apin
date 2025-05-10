service: {
  http: {
    servers: {
      main: {
        port: 8888
      }
    }
  }
  grpc: {
    servers: {
      main: {
        features: {
          logging: true
        }
      }
    }
  }
}
