package main

// Shared request/response types for the vessel daemon HTTP API.
// Defined here (no build tag) so both station_agent_windows.go and
// station_agent_other.go can reference them without duplication.

// agentRunContainerReq is sent to POST /container/run.
type agentRunContainerReq struct {
	App        string   `json:"app"`
	Image      string   `json:"image"`
	Command    []string `json:"command,omitempty"`
	Env        []string `json:"env,omitempty"` // KEY=VALUE pairs
	Volumes    []string `json:"volumes,omitempty"`
	ExtraHosts []string `json:"extra_hosts,omitempty"`
	NetMode    string   `json:"net_mode,omitempty"`
	Restart    string   `json:"restart,omitempty"`
	Port       int      `json:"port,omitempty"`
}

// agentRunContainerResp is the response from POST /container/run.
type agentRunContainerResp struct {
	ID   string `json:"id"`
	PID  int    `json:"pid"`
	Port int    `json:"port"`
	IP   string `json:"ip,omitempty"`
}

// agentProxyReq is sent to POST /proxy/start and POST /proxy/swap.
type agentProxyReq struct {
	App             string `json:"app"`
	Port            int    `json:"port,omitempty"`
	ActiveUpstream  string `json:"active_upstream"`
	StandbyUpstream string `json:"standby_upstream,omitempty"`
	ActiveSlot      string `json:"active_slot,omitempty"`
	StandbySlot     string `json:"standby_slot,omitempty"`
	TrafficMode     string `json:"traffic_mode,omitempty"`
	CookieName      string `json:"cookie_name,omitempty"`
	PublicHost      string `json:"public_host,omitempty"`
	ClearStandby    bool   `json:"clear_standby,omitempty"`
	ClearPublicHost bool   `json:"clear_public_host,omitempty"`
}


