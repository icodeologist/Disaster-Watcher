package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/controllers"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
	"github.com/icodeologist/disasterwatch/routes"
	pb "github.com/icodeologist/grpc-proto"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	address = ":50051"
)

func grpcConnection() (*grpc.ClientConn, error) {

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func main() {
	database.Connect()
	conn, err := grpcConnection()
	if err != nil {
		log.Printf("Could not establish the connection with grpc server %v\n", err)
	}
	defer conn.Close()
	client := pb.NewNotificationserviceClient(conn)

	go startRedis(client)

	RunDisasterAPI()
	fmt.Printf("The Disaster Notifier api has been started.\n")

}

func startRedis(client pb.NotificationserviceClient) {
	redisClient, ctx, err := database.ConnectToRedis()
	if err != nil {
		log.Fatal(err)
	}
	for {
		poppedReport, err := redisClient.LPop(ctx, "reportLists").Bytes()
		if err == redis.Nil {
			log.Printf("ERROR : cannot pop from emtpy stack")
			time.Sleep(2 * time.Second)
		} else if err != nil {
			log.Printf("Redis popping ERROR : %v\n", err)
		} else {
			var report models.Report
			err = json.Unmarshal(poppedReport, &report)
			if err != nil {
				log.Printf("ERROR :UNMARSHAL ERROR %v\n", err)
			}
			users, err := controllers.FindUserAffected(report)
			if err != nil {
				log.Printf("ERROR : %v\n", err)
			}
			usersNotificationRequests, err := controllers.MapEachUsersTONotificationEventInstance(users, report)
			if err != nil {
				log.Printf("ERROR : %v\n", err)
			}
			//first health check if notification service server is running
			HealthCheck(client)
			log.Print("HEALTH CHECK DONE")

			log.Print("Sending batch notification : START")
			SendBatchNotificationsToAffectedUsers(client, usersNotificationRequests, report)
			log.Print("Sending batch notification : END")
		}
	}
}

func RunDisasterAPI() {
	database.Connect()
	router := gin.Default()
	routes.SetUpRoutes(router)
	router.Run(":3000")
}

func HealthCheck(client pb.NotificationserviceClient) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	resp, err := client.Healthcheck(ctx, &pb.Healthcheckrequest{
		Serivice: "notification-service",
	})

	if err != nil {
		log.Fatalf("Health check failed %v\n", err)
	}
	if resp.Status == pb.Healthcheckresponse_serving {
		fmt.Printf("Notiifcation service is serving.\n")
	} else {
		log.Fatalf("Notification service is not serving.\n")
	}

}

//	func SendBatchNotificationsToAffectedUsers(client pb.NotificationserviceClient, notifEvents []models.NotificationEvent, report models.Report) {
//		ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
//		defer cancelFunc()
//
//		stream, err := client.Sendnotiificationacceptingdisasterapidata(ctx) // rpc method
//		if err != nil {
//			log.Fatalf("Failed to create a batch stream %v\n.", err)
//		}
//		fmt.Printf("Sending the batch of %v\n", len(notifEvents))
//		for _, user := range notifEvents {
//
//			// user id from uint to str
//			userIDStr := fmt.Sprintf("%v", user.User.ID)
//			userPhoneNumStr := fmt.Sprintf("%v", user.User.PhoneNumber)
//			// sending the request to  rpc method
//			request := &pb.Notificationrequestwithdata{
//				UserId:          userIDStr,
//				UserEmail:       user.User.Email,
//				UserPhoneNumber: userPhoneNumStr,
//				ReportType:      report.Type,
//				ReportLocation:  "Location need to be filled",
//				Timestamp:       timestamppb.Now(),
//				Type:            user.NotificationType,
//			}
//
//			if err := stream.Send(request); err != nil {
//				log.Printf("Failed to send the batch of notifications %v\n", err)
//			}
//			fmt.Printf("   📤 Queued: %s \n", userIDStr)
//			time.Sleep(200 * time.Millisecond)
//		}
//
//		resp, err := stream.CloseAndRecv()
//		if err != nil {
//			log.Fatalf("Failed to recieve the batch response %v\n.", err)
//		}
//		fmt.Printf("\n🎯 Batch Results:\n")
//		fmt.Printf("   Total Sent: %d\n", resp.TotalSent)
//		fmt.Printf("   Successful: %d\n", resp.Success)
//		fmt.Printf("   Failed: %d\n", resp.Failed)
//		fmt.Println("WE reached HERER")
//		fmt.Printf("Failed users %v\n", resp.FailedPhoneNums)
//		fmt.Printf("Failed users %v\n", resp.FailedUserEmails)
//
//		fmt.Println("Processed all the notifications.")
//	}
func SendBatchNotificationsToAffectedUsers(client pb.NotificationserviceClient, notifEvents []models.NotificationEvent, report models.Report) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	stream, err := client.Sendnotiificationacceptingdisasterapidata(ctx)
	if err != nil {
		log.Printf("Failed to create a batch stream: %v", err) // Changed from Fatalf to Printf
		return
	}

	fmt.Printf("Sending the batch of %v notifications\n", len(notifEvents))

	fmt.Print("USER ID IN SEND FUNCTION\n")
	for _, n := range notifEvents {
		fmt.Printf("%v\n", n)
		fmt.Printf("%v\n", n.UserID)
	}

	var failedSends int
	for i, events := range notifEvents {
		// More explicit type conversion
		userIDStr := strconv.FormatUint(uint64(events.UserID), 10)

		// getting user object from the db
		var user models.User
		res := database.DB.First(&user, "id=?", events.UserID)
		if res.RowsAffected == 0 {
			fmt.Print("NO users found\n")
		}
		if res.Error != nil {
			fmt.Printf("Error  while fetching user details : %v\n", res.Error.Error())
		}

		notifEvents[i].User = user

		userPhoneNumStr := strconv.FormatUint(uint64(user.PhoneNumber), 10)

		request := &pb.Notificationrequestwithdata{
			UserId:          userIDStr,
			UserEmail:       user.Email,
			UserPhoneNumber: userPhoneNumStr,
			ReportType:      report.Type,
			ReportLocation:  "Nil location for now", // Use actual location if available
			Timestamp:       timestamppb.Now(),
			Type:            events.NotificationType,
		}

		if err := stream.Send(request); err != nil {
			log.Printf("Failed to send notification to user %s: %v", userIDStr, err)
			failedSends++
			continue // Optionally continue with next user
		}
		fmt.Printf("User id %v\n", userIDStr)
		fmt.Printf("   📤 Queued: %s\n", userIDStr)
		// time.Sleep(200 * time.Millisecond) // Consider removing if not needed
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("Failed to receive batch response: %v", err)
		return
	}

	fmt.Printf("\n🎯 Batch Results:\n")
	fmt.Printf("   Total Attempted: %d\n", len(notifEvents))
	fmt.Printf("   Total Sent: %d\n", resp.TotalSent)
	fmt.Printf("   Successful: %d\n", resp.Success)
	fmt.Printf("   Failed: %d\n", resp.Failed)
	fmt.Printf("   Local Send Failures: %d\n", failedSends)
	fmt.Printf("Failed phone numbers: %v\n", resp.FailedPhoneNums)
	fmt.Printf("Failed emails: %v\n", resp.FailedUserEmails)
	fmt.Println("Processed all the notifications.")
}
