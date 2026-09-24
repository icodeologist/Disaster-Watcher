package api

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-gonic/gin"
)

// BindJSON decodes one request object and rejects fields that are not part of
// the endpoint's request DTO. This prevents clients from silently attempting
// to set server-owned fields such as report status or job state.
func BindJSON(c *gin.Context, destination any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request contains multiple JSON values")
		}
		return err
	}
	return nil
}
