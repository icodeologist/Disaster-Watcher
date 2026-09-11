package handler

import "github.com/icodeologist/disasterwatch/internal/models"

type Server struct {
	ReportChannel          chan models.ReportMessage
	AffectedUsersIdChannel chan models.NotificationDeliveryMessage
	VerificationChannel    chan models.VerificationMessage
}

// SO server struct act as a dependency bag
// NewWorkerServer() is clean way to access these dependencies when it  comes to use them
func NewWorkerServer(reportsChannel chan models.ReportMessage, deliveryChannel chan models.NotificationDeliveryMessage, verificationChannel chan models.VerificationMessage) *Server {
	return &Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: deliveryChannel,
		VerificationChannel:    verificationChannel,
	}
}
