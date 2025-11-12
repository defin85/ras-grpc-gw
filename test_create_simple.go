package main

import (
	"context"
	"fmt"
	"log"

	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	rasv1 "github.com/v8platform/protos/gen/ras/service/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Simple Test: CreateInfobase WITHOUT AuthenticateCluster ===")
	fmt.Println()

	// Connect to gateway
	conn, err := grpc.Dial("localhost:3002",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to gateway")
	fmt.Println()

	// Step 1: Get clusters
	fmt.Println("Step 1: GetClusters")
	rasClient := rasv1.NewClustersServiceClient(conn)

	ctx := context.Background()

	clustersResp, err := rasClient.GetClusters(ctx, &messagesv1.GetClustersRequest{})
	if err != nil {
		log.Fatalf("GetClusters failed: %v", err)
	}

	if len(clustersResp.Clusters) == 0 {
		log.Fatal("No clusters found")
	}

	clusterID := clustersResp.Clusters[0].GetUuid()
	fmt.Printf("✅ Found cluster: %s\n", clusterID)
	fmt.Println()

	// Step 2: CreateInfobase (WITHOUT authentication)
	fmt.Println("Step 2: CreateInfobase (no auth - cluster has no admin)")
	mgmtClient := pb.NewInfobaseManagementServiceClient(conn)

	createReq := &pb.CreateInfobaseRequest{
		ClusterId: clusterID,
		Name:      "TestBase_Simple",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb_simple",
	}

	fmt.Printf("Creating infobase: %s (DBMS: PostgreSQL)\n", createReq.Name)

	createResp, err := mgmtClient.CreateInfobase(ctx, createReq)
	if err != nil {
		fmt.Printf("\n❌ CreateInfobase FAILED: %v\n", err)
		fmt.Println()
		fmt.Println("If cluster has no admin configured, it should work without AuthenticateCluster")
		return
	}

	fmt.Printf("\n✅ ✅ ✅ SUCCESS! ✅ ✅ ✅\n\n")
	fmt.Printf("Infobase created successfully!\n")
	fmt.Printf("  ID: %s\n", createResp.InfobaseId)
	fmt.Printf("  Name: %s\n", createResp.Name)
	fmt.Printf("  Message: %s\n", createResp.Message)
	fmt.Println()
	fmt.Println("=== MY IMPLEMENTATION WORKS WITH RAS PROTOCOL! ===")
}
