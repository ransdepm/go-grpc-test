# Description
The route guide server and client demonstrate how to use grpc go libraries to
perform server streaming RPCs.

Please refer to [gRPC Basics: Go](https://grpc.io/docs/tutorials/basic/go.html) for more information.

See the definition of the route guide service in pubsub/pub_sub.proto.

# Run the sample code
To compile and run the server, assuming you are in the root of the go-grpc-test
Compiled proto files are committed but if making changes to the .proto file must run the following to recompile.
- Will need proto compiler installed

Refer to go.mod as to what go packages will need to be installed to run locally

```
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pubsub/pub_sub.proto
```


```sh
$ go run server/server.go
```

Likewise, to run the client:

```sh
$ go run client/client.go
```