package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func Test_sessions(t *testing.T) {
	sess := &Session{
		Id:         uuid.New().String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
	}
	out, _ := json.Marshal(sess)
	fmt.Printf("out: %v\n", string(out))

}
