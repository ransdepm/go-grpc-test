/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a simple gRPC server that demonstrates how to use gRPC-Go libraries
// to perform unary, client streaming, server streaming and full duplex RPCs.
//
// It implements the route guide service whose definition can be found in routeguide/route_guide.proto.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	pb "github.com/ransdepm/go-grpc-test/pubsub"
)

var (
	jsonDBFile             = flag.String("json_db_file", "", "A json file containing a list of features")
	port, errPort          = strconv.Atoi(goDotEnvVariable("PORT"))
	server_sleep, errSleep = strconv.Atoi(goDotEnvVariable("STREAM_SLEEP"))
)

type pubSubServer struct {
	pb.UnimplementedPubsubServer
	saveTransactions []*pb.SubscribeStreamResponse // read-only after initialized
}

type AuthResponse struct {
	AuthKey string `json:"auth_key"`
}

type TransactionResponse struct {
	Orders []Orders `json:"orders"`
}

type Orders struct {
	OrderId  string `json:"local_order_uuid"`
	VendorId int    `json:"vendor_id"`
	VenueId  int    `json:"venue_id"`
	TxTime   string `json:"created_at"`
}

func (s *pubSubServer) Subscribe(topic *pb.SubscribeRequest, stream pb.Pubsub_SubscribeServer) error {
	for true {
		var token = getAuth()
		var txs = getOrders(token)
		for _, transaction := range txs {
			if err := stream.Send(&transaction); err != nil {
				return err
			}
		}

		time.Sleep(time.Second * time.Duration(server_sleep))
	}
	return nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPubsubServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}

func newServer() *pubSubServer {
	s := &pubSubServer{}
	return s
}

func getAuth() string {
	req, err := http.NewRequest("POST", "https://api-gw.latest.sf.appetize-dev.com/auth/transactions", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("x-api-key", goDotEnvVariable("X_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	//responseDump, err := httputil.DumpResponse(resp, false)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(string(responseDump))

	var responseObject AuthResponse
	json.Unmarshal(responseData, &responseObject)
	//fmt.Println(responseObject.AuthKey)
	return responseObject.AuthKey
}

func getOrders(token string) []pb.SubscribeStreamResponse {
	var url string
	var start string
	var end string

	end = time.Now().UTC().Format(time.RFC3339)
	start = time.Now().Add(time.Second * time.Duration(-1*server_sleep)).UTC().Format(time.RFC3339)
	url = "https://api-gw.latest.sf.appetize-dev.com/transactions_api/orders?start_date=" + start + "&end_date=" + end + "&page=1"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	var bearer = "Bearer " + token
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	//responseDump, err := httputil.DumpResponse(resp, false)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(string(responseDump))

	var responseObject TransactionResponse
	json.Unmarshal(responseData, &responseObject)

	//if len(responseObject.Orders) > 0 {
	numOrders := strconv.FormatInt(int64(len(responseObject.Orders)), 10)
	fmt.Println("--Recieved " + numOrders + " orders between " + start + " and " + end + ".  Passing to client.")
	//}

	var txs = make([]pb.SubscribeStreamResponse, len(responseObject.Orders))
	for i, s := range responseObject.Orders {
		txs[i].Id = s.OrderId
		txs[i].Type = "sale"
		txs[i].Action = "order"
		txs[i].Timestamp = s.TxTime
		txs[i].ResourceUrl = "https://api-gw.latest.sf.appetize-dev.com/transactions_api/orders/" + s.OrderId
	}
	return txs
}

func VerifyToken(r string) (*jwt.Token, error) {
	tokenString := r
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func TokenValid(r string) error {
	token, err := VerifyToken(r)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
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
