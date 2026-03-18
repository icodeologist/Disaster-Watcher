package handler

import "github.com/icodeologist/disasterwatch/internal/models"

type Server struct {
	ReportChannel          chan models.Report
	AffectedUsersIdChannel chan uint
	VerificationChannel    chan VerificationMessage
}

// SO server struct act as a dependency bag
// NewWorkerServer() is clean way to access these dependencies when it  comes to use them
func NewWorkerServer(reportsChannel chan models.Report, affUserIdsChannel chan uint, verificationChannel chan VerificationMessage) *Server {
	return &Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: affUserIdsChannel,
		VerificationChannel:    verificationChannel,
	}
}
