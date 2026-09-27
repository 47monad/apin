module github.com/47monad/apin/integrationtest/lifecycle

go 1.27.0

require (
	github.com/47monad/apin v0.1.0
	github.com/47monad/apin/initrs/pginitr v0.1.0
	github.com/47monad/apin/initrs/rmqinitr v0.1.0
	github.com/jackc/pgx/v5 v5.7.5
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)

replace github.com/47monad/apin => ../..

replace github.com/47monad/apin/initrs/pginitr => ../../initrs/pginitr

replace github.com/47monad/apin/initrs/rmqinitr => ../../initrs/rmqinitr
