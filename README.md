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


If you want to run both the client and the server without regard for installing proper Go compilers, simly run docker compose
```
docker-compose up
```

However when deployed it will not run in such a manner as the server and client will be deployed independently

First create a netowrk for the two docker images to run on
```
docker network create grpc-net
```

To use docker to build and run the client
```
docker build -f Dockerfile_client -t grpc-client .
docker run --net grpc-net -p 50005:50005 grpc-client
```

To use docker to build and run the server
```
docker build -f Dockerfile_server -t grpc-server .
docker run --net grpc-net -p 8080:8080 grpc-server
```