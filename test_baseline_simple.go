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
	fmt.Println("=== Baseline Test: GetClusters + GetShortInfobases ===\n")

	conn, err := grpc.Dial("localhost:3002",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected\n")

	// Test GetClusters
	rasClient := rasv1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Calling GetClusters...")
	clustersResp, err := rasClient.GetClusters(ctx, &messagesv1.GetClustersRequest{})
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		return
	}

	fmt.Printf("✅ SUCCESS - %d cluster(s) found\n", len(clustersResp.Clusters))
	if len(clustersResp.Clusters) == 0 {
		return
	}

	clusterUUID := clustersResp.Clusters[0].GetUuid()
	fmt.Printf("   Using cluster: %s\n\n", clusterUUID)

	// Test GetShortInfobases
	fmt.Println("Calling GetShortInfobases...")
	infobasesClient := rasv1.NewInfobasesServiceClient(conn)

	infobasesResp, err := infobasesClient.GetShortInfobases(ctx, &messagesv1.GetInfobasesShortRequest{
		ClusterId: clusterUUID,
	})
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		return
	}

	fmt.Printf("✅ SUCCESS - %d infobase(s) found\n\n", len(infobasesResp.GetSessions()))
	fmt.Println("=== Baseline methods WORK! ===")
}
