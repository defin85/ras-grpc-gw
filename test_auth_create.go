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
	"google.golang.org/grpc/metadata"
)

func main() {
	fmt.Println("=== Integration Test: Authenticated CreateInfobase ===")
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

	// Step 2: Authenticate cluster (with empty credentials)
	fmt.Println("Step 2: AuthenticateCluster (empty credentials)")
	authClient := rasv1.NewAuthServiceClient(conn)

	var header metadata.MD
	authCtx := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{}))

	_, err = authClient.AuthenticateCluster(authCtx, &messagesv1.ClusterAuthenticateRequest{
		ClusterId: clusterID,
		User:      "", // No admin on cluster
		Password:  "",
	}, grpc.Header(&header))

	if err != nil {
		log.Fatalf("AuthenticateCluster failed: %v", err)
	}

	// Extract endpoint_id from response headers
	endpointIDs := header.Get("endpoint_id")
	if len(endpointIDs) == 0 {
		log.Fatal("No endpoint_id in response headers")
	}
	endpointID := endpointIDs[0]

	fmt.Printf("✅ Authenticated, endpoint_id: %s\n", endpointID)
	fmt.Println()

	// Step 3: CreateInfobase with endpoint_id
	fmt.Println("Step 3: CreateInfobase (with authentication)")
	mgmtClient := pb.NewInfobaseManagementServiceClient(conn)

	// Create context with endpoint_id
	authenticatedCtx := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"endpoint_id": endpointID,
	}))

	createReq := &pb.CreateInfobaseRequest{
		ClusterId: clusterID,
		Name:      "TestBase_Authenticated",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb_auth",
	}

	fmt.Printf("Creating infobase: %s (DBMS: PostgreSQL)\n", createReq.Name)

	createResp, err := mgmtClient.CreateInfobase(authenticatedCtx, createReq)
	if err != nil {
		fmt.Printf("\n❌ CreateInfobase FAILED: %v\n", err)
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
