package models

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/redis/go-redis/v9"
)

type Response struct {
	Value int
	Item  *Item
}

type SessionResponses struct {
	Responses []Response
}

type SkinnyResponse struct {
	ItemName string `json:"item_name"`
	Value    int    `json:"value"`
}

type SessionState struct {
	SessionId  string               `json:"session_id"`
	Energies   map[string][]float64 `json:"energies"`
	Excluded   []*string            `json:"excluded"`
	Responses  []*SkinnyResponse    `json:"responses"`
	Start      time.Time            `json:"start_time"`
	Expiration time.Time            `json:"expiration_time"`
}

func (s SessionState) ByteMarshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SessionStateByteUnmarshal(sessionState []byte) (*SessionState, error) {
	var ss SessionState
	dec := gob.NewDecoder(bytes.NewReader(sessionState))
	err := dec.Decode(&ss)
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func SessionStateFromId(sid string, rdb redis.Client, ctx *context.Context) (*SessionState, error) {
	val, err := rdb.Get(*ctx, sid).Bytes()
	if err != nil {
		return nil, err
	}
	rehyrdated, _ := SessionStateByteUnmarshal(val)
	return rehyrdated, nil
}
