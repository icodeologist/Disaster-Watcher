package utils

import (
	"os"
	"testing"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	godotenv.Load("/home/denzil/dev/Disaster-Watcher/backend/.env")
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestRecoverPendingJobs(t *testing.T) {

	job1 := models.Jobs{Status: "pending", Payload: []byte(`{}`)}
	job3 := models.Jobs{Status: "pending", Payload: []byte(`{}`)}
	job2 := models.Jobs{Status: "pending", Payload: []byte(`{}`)}

	db.DB.Create(&job1)
	db.DB.Create(&job2)
	db.DB.Create(&job3)

	ch := make(chan models.VerificationMessage, 500)
	RecoverPendingJobsFromDBOnStarting(ch)

	assert.GreaterOrEqual(t, len(ch), 3)

	recoverd := map[int64]bool{}
	for len(ch) > 0 {
		msg := <-ch
		recoverd[msg.JobID] = true
	}
	assert.True(t, recoverd[job1.Id])
	assert.True(t, recoverd[job2.Id])
	assert.True(t, recoverd[job3.Id])
}
