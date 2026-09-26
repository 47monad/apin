module github.com/47monad/apin/initrs/grpcinitr/integrationtest

go 1.24.0

replace github.com/47monad/apin/config => ../../../config

replace github.com/47monad/apin/initrs/grpcinitr => ..

require (
	github.com/47monad/apin/config v0.0.0
	github.com/47monad/apin/initrs/grpcinitr v0.0.0
)

require (
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/grpc v1.72.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
