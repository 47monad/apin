package interface

import (
  "github.com/47monad/apin/manifest/cue/common"
)

#GRPCServer: {
  port: *4748 | common.#Port
  features: {
    reflection: *false | bool
    healthCheck: *false | bool
    logging: *false | bool
  }
}

#GRPCClient: {
  address: *"" | string
}

#GRPC: {
  clients: [string]: #GRPCClient
  servers: [string]: #GRPCServer
}

grpc: #GRPC
