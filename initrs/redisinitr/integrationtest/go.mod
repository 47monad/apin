module github.com/47monad/apin/initrs/redisinitr/integrationtest

go 1.27.0

replace github.com/47monad/apin/initrs/redisinitr => ..

require (
	github.com/47monad/apin v0.1.0
	github.com/47monad/apin/initrs/redisinitr v0.1.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/47monad/apin => ../../..
