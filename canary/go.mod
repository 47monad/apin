module github.com/47monad/apin/canary

go 1.24.3

// Compatibility canary: imports every initr against the latest zaal.
// CI builds this so a zaal config change that breaks any initr fails fast.

require (
	github.com/47monad/apin/initrs/etcdinitr v0.0.0
	github.com/47monad/apin/initrs/grpcinitr v0.0.0
	github.com/47monad/apin/initrs/mongoinitr v0.0.0
	github.com/47monad/apin/initrs/pginitr v0.0.0
	github.com/47monad/apin/initrs/prominitr v0.0.0
	github.com/47monad/apin/initrs/rmqinitr v0.0.0
	github.com/47monad/apin/initrs/zapinitr v0.0.0
	github.com/47monad/zaal v0.1.3-0.20260916141141-b819246ff6dd
)

require (
	cuelabs.dev/go/oci/ociregistry v0.0.0-20241125120445-2c00c104c6e1 // indirect
	cuelang.org/go v0.12.0 // indirect
	github.com/47monad/apin v0.0.10-0.20250719080514-402b80009edb // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/apd/v3 v3.2.1 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/emicklei/proto v1.13.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus v1.0.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20241112170944-20d2c9ebc01d // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/rogpeppe/go-internal v1.13.2-0.20241226121412-a5dc8ff20d0a // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.etcd.io/etcd/api/v3 v3.6.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.0 // indirect
	go.etcd.io/etcd/client/v3 v3.6.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.2.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/oauth2 v0.27.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/grpc v1.72.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/47monad/apin => ../

replace github.com/47monad/apin/initrs/etcdinitr => ../initrs/etcdinitr

replace github.com/47monad/apin/initrs/grpcinitr => ../initrs/grpcinitr

replace github.com/47monad/apin/initrs/mongoinitr => ../initrs/mongoinitr

replace github.com/47monad/apin/initrs/pginitr => ../initrs/pginitr

replace github.com/47monad/apin/initrs/prominitr => ../initrs/prominitr

replace github.com/47monad/apin/initrs/rmqinitr => ../initrs/rmqinitr

replace github.com/47monad/apin/initrs/zapinitr => ../initrs/zapinitr
