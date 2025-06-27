package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func SendBatchNotificationsToAffectedUsers(client pb.NotificationserviceClient, notifEvents []models.NotificationEvent, report models.Report) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	stream, err := client.Sendnotiificationacceptingdisasterapidata(ctx) // rpc method
	if err != nil {
		log.Fatalf("Failed to create a batch stream %v\n.", err)
	}
	fmt.Printf("Sending the batch of %v\n", len(notifEvents))
	for _, user := range notifEvents {

		locationName, err := controllers.ReverseGeocoding(report.Latitude, report.Longitude)
		if err != nil {
			log.Printf("ERROR : reverese geocoding nil locaiton %v\n", err)
		}
		// sending the request to  rpc method
		request := &pb.Notificationrequestwithdata{
			UserId:          string(user.User.ID),
			UserEmail:       user.User.Email,
			UserPhoneNumber: string(user.User.PhoneNumber),
			ReportType:      report.Type,
			ReportLocation:  locationName,
			Timestamp:       timestamppb.Now(),
			Type:            user.NotificationType,
		}
		fmt.Printf("user request to rpc function : %v\n", request)

		if err := stream.Send(request); err != nil {
			log.Printf("Failed to send the batch of notifications %v\n", err)
		}
		fmt.Printf("   📤 Queued: %s (%s)\n", string(user.User.ID), user.Action)
		time.Sleep(200 * time.Millisecond)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Failed to recieve the batch response %v\n.", err)
	}
	fmt.Printf("\n🎯 Batch Results:\n")
	fmt.Printf("   Total Sent: %d\n", resp.TotalSent)
	fmt.Printf("   Successful: %d\n", resp.Success)
	fmt.Printf("   Failed: %d\n", resp.Failed)
	fmt.Println("WE reached HERER")
	fmt.Printf("Failed users %v\n", resp.FailedPhoneNums)
	fmt.Printf("Failed users %v\n", resp.FailedUserEmails)

	fmt.Println("Processed all the notifications.")
}
