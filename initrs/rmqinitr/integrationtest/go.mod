module github.com/47monad/apin/initrs/rmqinitr/integrationtest

go 1.24.0

replace github.com/47monad/apin/config => ../../../config

replace github.com/47monad/apin/initrs/rmqinitr => ..

require (
	github.com/47monad/apin/config v0.0.0
	github.com/47monad/apin/initrs/rmqinitr v0.0.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
