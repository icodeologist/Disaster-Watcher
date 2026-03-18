package handler

import "github.com/icodeologist/disasterwatch/internal/models"

type VerificationMessage struct {
	Report models.Report
	User   models.User
}
