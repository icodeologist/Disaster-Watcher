package utils

import (
	"context"
	"log"
	"time"

	"github.com/icodeologist/disasterwatch/models"
	pb "github.com/icodeologist/grpc-proto"
	"github.com/redis/go-redis/v9"
)

func ConnectToRedis() error {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost/6379",
		Password: "",
		DB:       0,
	})

	ping, err := client.Ping(ctx).Result()
	if err != nil {
		return err
	}
	log.Printf("Ping : %v\n", ping)

	// implement the list for the notificaiton service

	n1 := &models.NotificationEvent2{
		User:             "alice",
		Action:           "Report posted nearby",
		Message:          "WIthin 7 km there is flood going on. Please be careful",
		TimeStamp:        time.Now(),
		NotificationType: *pb.NotificationType_EMAIL.Enum(),
	}

	res, err := client.LPush(ctx, "Notification", n1).Result()
	if err != nil {
		return err
	}

	log.Print(res)
	return nil

}
