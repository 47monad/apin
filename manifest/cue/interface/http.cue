package interface

import (
  "github.com/47monad/apin/manifest/cue/common@v0"
)

#HTTPServer: {
  port: *4747 | common.#Port
}

#HTTP: {
  servers: [string]: #HTTPServer
}

http: #HTTP
