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
	fmt.Println("=== Baseline Test: GetClusters ===\n")

	conn, err := grpc.Dial("localhost:3002",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to gateway\n")

	rasClient := rasv1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Calling GetClusters...")

	clustersResp, err := rasClient.GetClusters(ctx, &messagesv1.GetClustersRequest{})
	if err != nil {
		fmt.Printf("❌ GetClusters FAILED: %v\n", err)
		fmt.Println("\nRAS server is NOT responding")
		return
	}

	fmt.Printf("✅ GetClusters SUCCESS\n")
	fmt.Printf("   Found %d cluster(s)\n", len(clustersResp.Clusters))
	for i, cl := range clustersResp.Clusters {
		fmt.Printf("   [%d] UUID: %s\n", i+1, cl.GetUuid())
	}
	fmt.Println("\nRAS server is working!")
}
