package main

import (
	"context"
	"fmt"
	"log"
	"time"

	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	rasv1 "github.com/v8platform/protos/gen/ras/service/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Testing with existing 1C cluster ===\n")

	conn, err := grpc.Dial("localhost:3002",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to ras-grpc-gw gateway\n")

	// Test 1: GetClusters
	fmt.Println("📊 Test 1: GetClusters")
	rasClient := rasv1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clustersResp, err := rasClient.GetClusters(ctx, &messagesv1.GetClustersRequest{})
	if err != nil {
		fmt.Printf("❌ GetClusters FAILED: %v\n\n", err)
		return
	}

	if len(clustersResp.Clusters) == 0 {
		fmt.Println("❌ No clusters found\n")
		return
	}

	fmt.Printf("✅ GetClusters SUCCESS - found %d cluster(s)\n", len(clustersResp.Clusters))
	
	clusterUUID := clustersResp.Clusters[0].GetUuid()
	fmt.Printf("   Cluster UUID: %s\n\n", clusterUUID)

	// Test 2: GetShortInfobases
	fmt.Println("📊 Test 2: GetShortInfobases")
	infobasesClient := rasv1.NewInfobasesServiceClient(conn)

	infobasesResp, err := infobasesClient.GetShortInfobases(ctx, &messagesv1.GetInfobasesShortRequest{
		ClusterId: clusterUUID,
	})
	if err != nil {
		fmt.Printf("❌ GetShortInfobases FAILED: %v\n\n", err)
		return
	}

	fmt.Printf("✅ GetShortInfobases SUCCESS - found %d infobase(s)\n", len(infobasesResp.GetInfobases()))
	
	for i, ib := range infobasesResp.GetInfobases() {
		fmt.Printf("   [%d] Name: %s, UUID: %s\n", i+1, ib.GetName(), ib.GetUuid())
	}
	
	fmt.Println("\n=== Baseline methods work! Ready to test CreateInfobase ===")
}
