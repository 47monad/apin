service: {
  name: "test"
  mode: "debug"
  // logging: {
  //   level: "error"
  // }
  http: {
    servers: main: {
      port: 8787
    }
  }
}
