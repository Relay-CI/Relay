//go:build !windows

package main

// On non-Windows platforms vessel runs natively (no WSL2 layer).  All
// agent-gated paths in runtime_vessel.go are guarded by runtime.GOOS ==
// "windows", so these stubs satisfy the compiler but are never called.

import "io"

func getStationAgent() (*stationAgent, error) { return nil, nil }
func resetStationAgent()                     { /* no-op: Windows-only */ }
func stationAgentHealthy() bool              { return false }
func wslAvailableForAgent() bool            { return false }
func startStationAgentBackground()           { /* no-op: Windows-only */ }

type stationAgent struct{}

func (a *stationAgent) BuildDockerfile(_, _, _ string, _ io.Writer) (*stationManifest, error) {
	return nil, nil
}
func (a *stationAgent) GetManifest(_ string) (*stationManifest, error) { return nil, nil }
func (a *stationAgent) SnapshotExists(_ string) bool                  { return false }
func (a *stationAgent) DeleteSnapshot(_ string)             { /* no-op: Windows-only */ }
func (a *stationAgent) PruneSnapshots(_ string, _ []string) { /* no-op: Windows-only */ }
func (a *stationAgent) RunContainer(_ agentRunContainerReq) (*agentRunContainerResp, error) {
	return nil, nil
}
func (a *stationAgent) StopApp(_ string)                        { /* no-op: Windows-only */ }
func (a *stationAgent) ProxyStart(_ agentProxyReq) (int, error) { return 0, nil }
func (a *stationAgent) ProxySwap(_ agentProxyReq) error         { return nil }
func (a *stationAgent) ProxyStop(_ string)                      { /* no-op: Windows-only */ }

