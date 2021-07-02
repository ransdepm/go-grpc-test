// Package main implements a simple gRPC client that implements receiving data from server side streaming.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/joho/godotenv"

	pb "github.com/ransdepm/go-grpc-test/pubsub"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", goDotEnvVariable("SERVER_HOST")+":"+goDotEnvVariable("PORT"), "The server address in the format of host:port")
)

func grabTransactions(client pb.PubsubClient) {
	in := &pb.SubscribeRequest{
		TopicName: "orders",
	}

	stream, err := client.Subscribe(context.Background(), in)
	if err != nil {
		log.Fatalf("%v.Subscribe(_) = _, %v", client, err)
	}
	for {
		transaction, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.Subscribe(_) = _, %v", client, err)
		}
		fmt.Println(prettyPrint(transaction))
	}
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	//grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), grpc.WithPerRPCCredentials(jwtCreds))
	opts = append(opts, grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()))
	opts = append(opts, grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()))

	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewPubsubClient(conn)

	grabTransactions(client)
}

func CreateToken(userid uint64) (string, error) {
	var err error
	//Creating Access Token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userid
	atClaims["exp"] = time.Now().Add(time.Minute * 60).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte(goDotEnvVariable("ACCESS_SECRET")))
	if err != nil {
		return "", err
	}
	return token, nil
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	var x = "--------------------------------------------------------------------------\n" +
		string(s) +
		"\n--------------------------------------------------------------------------\n"
	return string(x)
}

// use godot package to load/read the .env file and
// return the value of the key
func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}
