package db

import "github.com/47monad/apin/manifest/cue/common"

#Pool: {
  maxConns?:            int & >=1
  minConns?:            int & >=0
  maxConnLifetime?:     int & >0
  maxConnIdleTime?:     int & >0
  healthCheckInterval?: int & >0
}

#Postgres: {
  uri?:         string
  host?:        string
  port?:        common.#Port
  username?:    string
  password?:    string
  dbName?:      string
  sslMode?:     "disable" | "allow" | "prefer" | "require" | "verify-ca" | "verify-full"
  appName?:     string
  connTimeout?: int & >0
  mode:         *"pool" | "single"
  pool?:        #Pool
}

postgres: #Postgres
