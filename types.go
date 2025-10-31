package xtop

type SessionDetails struct {
	AuthID     string `json:"authid"`
	AuthRole   string `json:"authrole"`
	Serializer string `json:"serializer"`
	SessionID  uint64 `json:"sessionID"`
}

type RouterStats struct {
	CPUUsage       float64 `json:"cpu_usage"`
	ReservedMemory uint64  `json:"res_memory"`
	Uptime         int64   `json:"uptime"`
}
