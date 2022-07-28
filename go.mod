module github.com/leicc520/go-micro-grpc

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	github.com/leicc520/go-orm v1.0.0
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b
	google.golang.org/grpc v1.48.0
)

replace github.com/leicc520/go-orm v1.0.0 => ../go-orm
