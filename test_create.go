package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Testing CreateInfobase ===\n")

	conn, err := grpc.Dial("localhost:3002",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to gateway\n")

	mgmtClient := pb.NewInfobaseManagementServiceClient(conn)

	createReq := &pb.CreateInfobaseRequest{
		ClusterId: "00000000-0000-0000-0000-000000000000",
		Name:      "TestBase",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Sending CreateInfobase request...\n")
	fmt.Printf("  Name: %s\n", createReq.Name)
	fmt.Printf("  DBMS: %s\n", createReq.Dbms)
	fmt.Printf("  Server: %s\n\n", createReq.DbServer)

	createResp, err := mgmtClient.CreateInfobase(ctx, createReq)
	if err != nil {
		fmt.Printf("❌ RESULT: CreateInfobase FAILED\n")
		fmt.Printf("   Error: %v\n\n", err)
		fmt.Println("Conclusion: My implementation does NOT work with RAS Binary Protocol")
		return
	}

	fmt.Printf("✅ RESULT: CreateInfobase SUCCESS!\n")
	fmt.Printf("   Infobase ID: %s\n")
	fmt.Printf("   Name: %s\n", createResp.Name)
	fmt.Printf("   Message: %s\n\n", createResp.Message)
	fmt.Println("Conclusion: My implementation WORKS with RAS Binary Protocol!")
}
