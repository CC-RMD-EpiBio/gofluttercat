package models

import (
	"bytes"
	"encoding/gob"
	"time"
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
	SessionId  string            `json:"session_id"`
	Energy     []float64         `json:"energy"`
	Excluded   []*string         `json:"excluded"`
	Responses  []*SkinnyResponse `json:"responses"`
	Start      time.Time         `json:"start_time"`
	Expiration time.Time         `json:"expiration_time"`
}

func (s SessionState) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(s)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Unmarshal(sessionState []byte) SessionState {
	var ss SessionState
	dec := gob.NewDecoder(bytes.NewReader(sessionState))
	err := dec.Decode(&ss)
	if err != nil {
		panic(err)
	}
	return ss
}
