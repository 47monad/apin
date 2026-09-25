package interface

import (
  "github.com/47monad/apin/manifest/cue/common"
)

#HTTPServer: {
  port: *4747 | common.#Port
}

#HTTP: {
  servers: [string]: #HTTPServer
}

http: #HTTP
