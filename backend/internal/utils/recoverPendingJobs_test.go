package utils

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	godotenv.Load("/home/denzil/dev/Disaster-Watcher/backend/.env")
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestRecoverUnfinishedJobs(t *testing.T) {
	originalDB := db.DB
	tx := originalDB.Begin()
	require.NoError(t, tx.Error)
	db.DB = tx
	t.Cleanup(func() {
		tx.Rollback()
		db.DB = originalDB
	})

	pendingJob := models.Jobs{Status: "pending", Payload: []byte(`{}`)}
	processingJob := models.Jobs{Status: "processing", Payload: []byte(`{}`), Started_at: time.Now()}
	deliveryJob := models.Jobs{Status: "processing", Payload: []byte(`{}`), Started_at: time.Now()}

	for _, job := range []*models.Jobs{&pendingJob, &processingJob, &deliveryJob} {
		require.NoError(t, db.DB.Create(job).Error)
	}

	delivery := models.NotificationDelivery{
		IdempotencyKey: fmt.Sprintf("recovery-test/%d", deliveryJob.Id),
		JobID:          deliveryJob.Id,
		UserID:         uint(deliveryJob.Id),
		RecipientEmail: "recovery@example.com",
		Status:         models.DeliveryStatusPending,
	}
	require.NoError(t, db.DB.Create(&delivery).Error)

	verificationChannel := make(chan models.VerificationMessage, 500)
	require.NoError(t, RecoverUnfinishedJobs(context.Background(), verificationChannel))

	recovered := map[int64]bool{}
	for len(verificationChannel) > 0 {
		message := <-verificationChannel
		recovered[message.JobID] = true
	}

	assert.True(t, recovered[pendingJob.Id])
	assert.True(t, recovered[processingJob.Id])
	assert.False(t, recovered[deliveryJob.Id])

	var recoveredProcessingJob models.Jobs
	require.NoError(t, db.DB.First(&recoveredProcessingJob, processingJob.Id).Error)
	assert.Equal(t, "pending", recoveredProcessingJob.Status)
}
