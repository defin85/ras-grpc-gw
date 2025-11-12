package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	rasv1 "github.com/v8platform/protos/gen/ras/service/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Integration Test: ras-grpc-gw ===")
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

	fmt.Println("Connected to ras-grpc-gw on localhost:3002")
	fmt.Println()

	// Test 1: GetClusters (baseline)
	fmt.Println("Test 1: GetClusters (baseline)")
	rasClient := rasv1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clustersResp, err := rasClient.GetClusters(ctx, &messagesv1.GetClustersRequest{})
	if err != nil {
		log.Fatalf("GetClusters failed: %v", err)
	}

	if len(clustersResp.Clusters) == 0 {
		log.Fatal("No clusters found")
	}

	clusterID := clustersResp.Clusters[0].ClusterId
	fmt.Printf("Found %d cluster(s), using ID: %s\n", len(clustersResp.Clusters), clusterID)
	fmt.Println()

	// Test 2: GetShortInfobases (baseline)
	fmt.Println("Test 2: GetShortInfobases (baseline)")
	infobasesClient := rasv1.NewInfobasesServiceClient(conn)

	infobasesResp, err := infobasesClient.GetShortInfobases(ctx, &messagesv1.GetInfobasesShortRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		log.Fatalf("GetShortInfobases failed: %v", err)
	}

	fmt.Printf("Found %d infobase(s)\n", len(infobasesResp.Infobases))
	fmt.Println()

	// Test 3: CreateInfobase (наша реализация!)
	fmt.Println("Test 3: CreateInfobase (OUR IMPLEMENTATION)")
	mgmtClient := pb.NewInfobaseManagementServiceClient(conn)

	createReq := &pb.CreateInfobaseRequest{
		ClusterId: clusterID,
		Name:      "TestBase_Integration",
		Dbms:      pb.DBMSType_DBMS_TYPE_FILE,
		DbServer:  "",
		DbName:    "C:\Temp\TestBase_Integration",
	}

	fmt.Printf("Creating infobase: %s (DBMS: FILE)\n", createReq.Name)

	createResp, err := mgmtClient.CreateInfobase(ctx, createReq)
	if err != nil {
		fmt.Printf("\nRESULT: CreateInfobase FAILED: %v\n", err)
		fmt.Println("My implementation does NOT work with RAS protocol")
		return
	}

	fmt.Printf("\nRESULT: CreateInfobase SUCCESS!\n")
	fmt.Printf("Infobase ID: %s\n", createResp.InfobaseId)
	fmt.Printf("My implementation WORKS with RAS protocol!\n")
}
