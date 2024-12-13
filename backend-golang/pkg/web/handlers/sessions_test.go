package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/models"
	"github.com/google/uuid"
	"github.com/mederrata/ndvek"
	"github.com/redis/go-redis/v9"
)

func Test_sessions(t *testing.T) {
	data := ndvek.Linspace(-10, 10, 400)
	energies := make(map[string][]float64, 0)
	energies["A"] = data
	sess := &models.SessionState{
		SessionId:  uuid.New().String(),
		Start:      time.Now(),
		Expiration: time.Now().Local().Add(time.Hour * time.Duration(24)),
		Energies:   energies,
	}

	out, _ := json.Marshal(sess)
	fmt.Printf("out: %v\n", string(out))
	bout, _ := sess.ByteMarshal()
	fmt.Printf("bout: %v\n", bout)
	rehyrdated, _ := models.SessionStateByteUnmarshal(bout)
	fmt.Printf("rehyrdated: %v\n", rehyrdated)

	conf := config.GetConfig()
	rdb := redis.NewClient(&redis.Options{
		Addr: conf.Redis.Host + ":" + conf.Redis.Port,
	})

	ctx := context.Background()
	err := rdb.Ping(ctx).Err()
	if err != nil {
		fmt.Println("failed to connect to redis: %w", err)
	}
	log.Printf("Connected to Redis at ")

	defer func() {
		if err := rdb.Close(); err != nil {
			fmt.Println("failed to close redis", err)
		}
	}()
	stus := rdb.Set(ctx, sess.SessionId, bout, sess.Expiration.Sub(time.Now()))
	fmt.Printf("stus: %v\n", stus)

}
