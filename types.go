package xtop

const (
	StatusIdle    = "Idle"
	StatusRunning = "Running"
	StatusOffline = "Offline"
)

type SessionInfo struct {
	AuthID     string `json:"authid"`
	AuthRole   string `json:"authrole"`
	Serializer string `json:"serializer"`
	SessionID  uint64 `json:"sessionID"`
}
