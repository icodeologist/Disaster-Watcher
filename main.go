package main

import (
	// "context"
	"fmt"
	// "log"
	// "time"
	//
	// "github.com/gin-gonic/gin"
	// "github.com/icodeologist/disasterwatch/models"
	"github.com/icodeologist/disasterwatch/utils"
	// "github.com/icodeologist/disasterwatch/routes"
	// pb "github.com/icodeologist/grpc-proto"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
	// "google.golang.org/protobuf/types/known/timestamppb"
)

// const (
//
//	address = ":50051"
//
// )
func main() {
	// // grpc logic
	// conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	//
	//	if err != nil {
	//		log.Fatal("Could not establish the connection with grpc server %v\n", err)
	//	}
	//
	// defer conn.Close()
	// client := pb.NewNotificationServiceClient(conn)
	// RunDisasterAPI()
	// fmt.Println("The Disaster Notifier api has been starter.\n")
	//
	// // get the effected users
	// // then call batch send notification
	err := utils.ConnectToRedis()
	if err != nil {
		fmt.Println(err)
	}

}

//
// func RunDisasterAPI() {
// 	database.Connect()
// 	router := gin.Default()
// 	routes.SetUpRoutes(router)
// 	router.Run(":3000")
// }
//
// func healthCheck(client pb.NotificationServiceClient) {
// 	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancelFunc()
//
// 	resp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{
// 		Serivice: "notification-service",
// 	})
//
// 	if err != nil {
// 		log.Fatalf("Health check failed %v\n", err)
// 	}
// 	if resp.Status == pb.HealthCheckResponse_SERVING {
// 		fmt.Println("Notiifcation service is serving.\n")
// 	} else {
// 		log.Fatalf("Notification service is not serving.\n")
// 	}
//
// }
//
// // sends users who are effected with disasters
// // batch notifiacton service
// func SendBatchNotificationsToAffectedUsers(client pb.NotificationServiceClient, affectedUsers []models.NotificationEvent) {
// 	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancelFunc()
//
// 	stream, err := client.SendBatchNotification(ctx)
// 	if err != nil {
// 		log.Fatalf("Failed to create a batch stream %v\n.", err)
// 	}
// 	fmt.Printf("Sending the batch of %v\n", len(affectedUsers))
// 	//send all the notifications
// 	//
// 	for _, user := range affectedUsers {
// 		request := &pb.NotificationRequest{
// 			UserId:    string(user.User.ID),
// 			Action:    user.Action,
// 			Message:   user.Message,
// 			Timestamp: timestamppb.Now(),
// 			Type:      user.NotificationType,
// 		}
//
// 		if err := stream.Send(request); err != nil {
// 			log.Printf("Failed to send the batch of notifications %v\n", err)
// 		}
// 		fmt.Printf("   📤 Queued: %s (%s)\n", string(user.User.ID), user.Action)
// 		// stimulating the time required tosend the notifications
// 		// TODO: can add go routines here???
// 		time.Sleep(200 * time.Millisecond)
// 	}
//
// 	//close the stream and get the response fromit
//
// 	resp, err := stream.CloseAndRecv()
// 	if err != nil {
// 		log.Fatalf("Failed to recieve the batch response %v\n.", err)
// 	}
// 	fmt.Printf("\n🎯 Batch Results:\n")
// 	fmt.Printf("   Total Sent: %d\n", resp.TotalSent)
// 	fmt.Printf("   Successful: %d\n", resp.Success)
// 	fmt.Printf("   Failed: %d\n", resp.Failed)
//
// 	if len(resp.FailedUserId) > 0 {
// 		fmt.Printf("   Failed User IDs: %v\n", resp.FailedUserId)
// 	}
//
// 	fmt.Println("Processed all the notifications.")
// }
//
