// Package main implements a simple gRPC client that implements receiving data from server side streaming.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/joho/godotenv"
	"pack.ag/amqp"

	pb "github.com/ransdepm/go-grpc-test/pubsub"
	"google.golang.org/grpc"
)

var (
	serverAddr = flag.String("server_addr", goDotEnvVariable("SERVER_HOST")+":"+goDotEnvVariable("PORT"), "The server address in the format of host:port")
)

func HandleTransactions(client pb.PubsubClient, mq_session *amqp.Session) {
	in := &pb.SubscribeRequest{
		TopicName: "orders",
	}

	//Create gRPC stream connection with server
	stream, err := client.Subscribe(context.Background(), in)
	if err != nil {
		log.Fatalf("%v.Subscribe(_) = _, %v", client, err)
	}

	//Create MQ send connection to populate MQ
	sender, err := mq_session.NewSender(
		amqp.LinkTargetAddress(goDotEnvVariable("MQ_QUEUE")),
	)
	if err != nil {
		log.Fatal("Creating sender link:", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	for {
		transaction, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.Subscribe(_) = _, %v", client, err)
		}
		//Add transaction to MQ.  For now print it out.
		log.Println(prettyPrint(transaction))

		// Send message
		err = sender.Send(ctx, amqp.NewMessage(jsonByteArray(transaction)))
		if err != nil {
			log.Fatal("Sending message:", err)
		}
	}

	sender.Close(ctx)
	cancel()
}

func main() {
	flag.Parse()

	//Create client connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	//TBD: use JWT credentials when making connection
	//grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")), grpc.WithPerRPCCredentials(jwtCreds))

	//Add options for default interceptors that will allow for client retry 5 times when connections lost.
	//TBD: Add more robust retry method
	opts = append(opts, grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()))
	opts = append(opts, grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()))

	opts = append(opts, grpc.WithBlock())
	log.Printf("Attempting to connect to %v", *serverAddr)
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewPubsubClient(conn)

	mq_client, err := amqp.Dial(goDotEnvVariable("MQ_INSTANCE"),
		amqp.ConnSASLPlain(goDotEnvVariable("MQ_USERNAME"), goDotEnvVariable("MQ_PW")),
	)
	if err != nil {
		log.Fatal("Dialing AMQP server:", err)
	}
	defer mq_client.Close()

	mq_session, err := mq_client.NewSession()
	if err != nil {
		log.Fatal("Creating AMQP session:", err)
	}

	HandleTransactions(client, mq_session)
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

func jsonByteArray(i interface{}) []byte {
	s, _ := json.Marshal(i)
	return []byte(string(s))
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
