package main

import (
	"context"
	"fmt"
	"log"

	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	rasv1 "github.com/v8platform/protos/gen/ras/service/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("=== Test: GetShortInfobases via Docker Gateway (port 9999) ===")
	fmt.Println()

	conn, err := grpc.Dial("localhost:9999",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to Docker gateway (port 9999)")
	fmt.Println()

	// Get clusters
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

	// Get infobases
	fmt.Println("Step 2: GetShortInfobases")
	infobasesClient := rasv1.NewInfobasesServiceClient(conn)

	infobasesResp, err := infobasesClient.GetShortInfobases(ctx, &messagesv1.GetInfobasesShortRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		fmt.Printf("❌ GetShortInfobases FAILED: %v\n", err)
		return
	}

	fmt.Printf("✅ GetShortInfobases SUCCESS\n")
	infobases := infobasesResp.GetSessions()
	fmt.Printf("   Found %d infobase(s)\n", len(infobases))
	for i, ib := range infobases {
		fmt.Printf("   [%d] Name: %s, UUID: %s\n", i+1, ib.GetName(), ib.GetUuid())
	}
}
