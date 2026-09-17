package cue

import (
  "github.com/47monad/apin/manifest/cue/broker"
  "github.com/47monad/apin/manifest/cue/db"
  "github.com/47monad/apin/manifest/cue/log"
  "github.com/47monad/apin/manifest/cue/interface"
  "github.com/47monad/apin/manifest/cue/monitoring"
)

#Env: "production" | "staging" | "dev"
#Mode: "normal" | "debug"

#Schema: {
  name: *"go-app" | string
  title: *"Go App" | string
  version: *"1.0.0" | string
  host: *"127.0.0.1" | string
  env: *"dev" | #Env
  mode: *"normal" | #Mode
  logging: log.config
  mongodb?: db.mongodb
  postgres?: db.postgres
  etcd?: db.etcd
  rabbitmq?: broker.rabbitmq
  prometheus?: monitoring.prometheus
  grpc?: interface.grpc
  http?: interface.http
}

service: #Schema

