package broker

#RabbitMQ: {
  uri: *"" | string
  minRetryInterval: *1 | >0
  maxRetryInterval: *30 | >1
}

rabbitmq: #RabbitMQ
