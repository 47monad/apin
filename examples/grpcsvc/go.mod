module github.com/47monad/apin/examples/grpcsvc

go 1.24.3

require (
	github.com/47monad/apin v0.0.10-0.20250719080514-402b80009edb
	github.com/47monad/apin/config v0.0.0
	github.com/47monad/apin/initrs/grpcinitr v0.0.0
	github.com/47monad/apin/initrs/httpinitr v0.0.0
	github.com/47monad/apin/initrs/pginitr v0.0.0
	github.com/47monad/apin/initrs/zapinitr v0.0.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/grpc v1.72.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/47monad/apin => ../..

replace github.com/47monad/apin/config => ../../config

replace github.com/47monad/apin/initrs/grpcinitr => ../../initrs/grpcinitr

replace github.com/47monad/apin/initrs/httpinitr => ../../initrs/httpinitr

replace github.com/47monad/apin/initrs/pginitr => ../../initrs/pginitr

replace github.com/47monad/apin/initrs/zapinitr => ../../initrs/zapinitr
