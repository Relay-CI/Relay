//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	bridgeName   = "station0"
	bridgeSubnet = "10.88.0.0/16"
	bridgeIP     = "10.88.0.1"
	containerGW  = "10.88.0.1"
)

// ─── state ────────────────────────────────────────────────────────────────────

type netRecord struct {
	IP       string `json:"ip"`        // assigned container IP, e.g. "10.88.0.2"
	VethHost string `json:"veth_host"` // bridge-side interface, e.g. "veth-a1b2c3d4"
	VethPeer string `json:"veth_peer"` // container-side interface, e.g. "vethp-a1b2c3d4"
}

type networkState struct {
	// NextIdx is the sequential counter for IP assignment.
	// index 2 → 10.88.0.2, index 258 → 10.88.1.2, etc.
	NextIdx    int                   `json:"next_idx"`
	Containers map[string]*netRecord `json:"containers"`
}

var netMu sync.Mutex

func netStatePath() string {
	return filepath.Join(stateBaseDir(), "network.json")
}

func loadNetState() *networkState {
	data, err := os.ReadFile(netStatePath())
	if err != nil {
		return &networkState{NextIdx: 2, Containers: make(map[string]*netRecord)}
	}
	var s networkState
	if json.Unmarshal(data, &s) != nil {
		return &networkState{NextIdx: 2, Containers: make(map[string]*netRecord)}
	}
	if s.Containers == nil {
		s.Containers = make(map[string]*netRecord)
	}
	if s.NextIdx < 2 {
		s.NextIdx = 2
	}
	return &s
}

func saveNetState(s *networkState) error {
	_ = os.MkdirAll(stateBaseDir(), 0755)
	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(netStatePath(), data, 0644)
}

func idxToIP(idx int) string {
	return fmt.Sprintf("10.88.%d.%d", idx/256, idx%256)
}

// ─── IP allocation ────────────────────────────────────────────────────────────

// allocOrReuseContainerIP returns the existing IP for key on restart, or
// allocates the next available address from 10.88.0.0/16 on first run.
func allocOrReuseContainerIP(key string) (string, error) {
	netMu.Lock()
	defer netMu.Unlock()
	s := loadNetState()
	if rec, ok := s.Containers[key]; ok {
		return rec.IP, nil // stable across restarts
	}
	if s.NextIdx > 65534 {
		return "", fmt.Errorf("IP pool exhausted (10.88.0.0/16 full)")
	}
	ip := idxToIP(s.NextIdx)
	s.NextIdx++
	s.Containers[key] = &netRecord{IP: ip}
	return ip, saveNetState(s)
}

// ─── public API ───────────────────────────────────────────────────────────────

// prepareNetworkEnv is called in cmdRun before doSpawn. It allocates a stable
// IP for the container and populates rec.Env with the CONTAINER_NET_* vars
// that containerInit reads to configure eth0 inside the new network namespace.
func prepareNetworkEnv(rec *ContainerRecord) error {
	// Auto-enable bridge isolation when running as root and no explicit network
	// mode was requested. On WSL2, station always runs as root so containers get
	// isolated network namespaces by default, matching Docker's behavior.
	if rec.NetMode == "" && os.Getuid() == 0 {
		rec.NetMode = "bridge"
	}
	if rec.NetMode != "bridge" {
		return nil
	}
	if os.Getuid() != 0 {
		// Bridge networking needs root for veth pairs, ip-link, and iptables.
		// When running rootless, degrade to host networking: the container
		// shares the host network namespace and binds directly to the
		// allocated port — sufficient for the relay web-app use case.
		fmt.Fprintf(os.Stderr,
			"warn: --net bridge requires root; falling back to host networking (rootless)\n"+
				"      to use bridge networking run: sudo station run --net bridge ...\n")
		rec.NetMode = "" // host networking — no CLONE_NEWNET, no veth
		return nil
	}
	rec.NetworkKey = containerNetworkKey(rec.App, rec.ID)
	ip, err := allocOrReuseContainerIP(rec.NetworkKey)
	if err != nil {
		return err
	}
	rec.IP = ip
	short := rec.ID
	if len(short) > 8 {
		short = short[:8]
	}
	if rec.Env == nil {
		rec.Env = make(map[string]string)
	}
	rec.Env["CONTAINER_NET_MODE"] = "bridge"
	rec.Env["CONTAINER_NET_IP"] = ip
	rec.Env["CONTAINER_NET_GW"] = containerGW
	rec.Env["CONTAINER_NET_PEER"] = "vethp-" + short
	return nil
}

// setupContainerNetwork is called from doSpawn after the child process starts.
// It creates a veth pair, attaches the host end to the station0 bridge, and
// moves the peer end into the container's network namespace (pid).
func setupContainerNetwork(rec *ContainerRecord, pid int) error {
	if err := ensureBridge(); err != nil {
		return err
	}
	short := rec.ID
	if len(short) > 8 {
		short = short[:8]
	}
	vethHost := "veth-" + short
	vethPeer := "vethp-" + short
	key := containerNetworkKey(rec.App, rec.ID)

	// Remove any stale veth pair from a prior run with this container short-ID.
	_ = ipRun("link", "del", vethHost)

	if err := ipRun("link", "add", vethHost, "type", "veth", "peer", "name", vethPeer); err != nil {
		return fmt.Errorf("create veth pair: %w", err)
	}
	if err := ipRun("link", "set", vethHost, "master", bridgeName); err != nil {
		return fmt.Errorf("attach veth to bridge: %w", err)
	}
	if err := ipRun("link", "set", vethHost, "up"); err != nil {
		return fmt.Errorf("bring up veth host end: %w", err)
	}
	actualPeer, err := actualVethPeerName(vethHost)
	if err != nil {
		_ = ipRun("link", "del", vethHost)
		return err
	}
	// Move the peer end into the container's network namespace.
	pidStr := fmt.Sprintf("%d", pid)
	if err := ipRun("link", "set", actualPeer, "netns", pidStr); err != nil {
		_ = ipRun("link", "del", vethHost) // clean up host veth on failure
		return fmt.Errorf("move veth peer to container netns: %w", err)
	}
	if err := configureMovedPeer(pidStr, actualPeer, rec.IP, containerGW); err != nil {
		_ = ipRun("link", "del", vethHost)
		return err
	}

	// Persist veth names for teardown on stop.
	netMu.Lock()
	s := loadNetState()
	if netRec, ok := s.Containers[key]; ok {
		netRec.VethHost = vethHost
		netRec.VethPeer = actualPeer
	} else {
		s.Containers[key] = &netRecord{IP: rec.IP, VethHost: vethHost, VethPeer: actualPeer}
	}
	_ = saveNetState(s)
	netMu.Unlock()
	// Forward localhost:PORT → containerIP:PORT so that host processes (e.g. the
	// station proxy daemon) can reach the container on its allocated port without
	// any changes to relayd or the proxy configuration.
	if err := addPortForward(rec.Port, rec.IP); err != nil {
		fmt.Fprintf(os.Stderr, "warn: port-forward %d → %s: %v\n", rec.Port, rec.IP, err)
	}
	return nil
}

func actualVethPeerName(vethHost string) (string, error) {
	out, err := exec.Command("ip", "-o", "link", "show", "dev", vethHost).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect veth peer for %s: %w: %s", vethHost, err, strings.TrimSpace(string(out)))
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return "", fmt.Errorf("inspect veth peer for %s: unexpected output %q", vethHost, strings.TrimSpace(string(out)))
	}
	nameField := strings.TrimSuffix(fields[1], ":")
	if idx := strings.Index(nameField, "@"); idx >= 0 && idx+1 < len(nameField) {
		return nameField[idx+1:], nil
	}
	return "", fmt.Errorf("inspect veth peer for %s: missing peer in %q", vethHost, nameField)
}

func configureMovedPeer(pidStr, peerName, ip, gw string) error {
	run := func(args ...string) error {
		cmdArgs := append([]string{"-t", pidStr, "-n", "ip"}, args...)
		out, err := exec.Command("nsenter", cmdArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("nsenter ip %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	if err := run("link", "set", "lo", "up"); err != nil {
		return err
	}
	if peerName != "eth0" {
		if err := run("link", "set", peerName, "name", "eth0"); err != nil {
			return err
		}
	}
	if err := run("link", "set", "eth0", "up"); err != nil {
		return err
	}
	if err := run("addr", "add", ip+"/16", "dev", "eth0"); err != nil && !strings.Contains(err.Error(), "File exists") {
		return err
	}
	if err := run("route", "add", "default", "via", gw); err != nil && !strings.Contains(err.Error(), "File exists") {
		return err
	}
	return nil
}

// teardownContainerNetwork removes the veth pair for a container on stop.
// It is a no-op if the interface has already been cleaned up by the kernel.
func teardownContainerNetwork(rec *ContainerRecord) {
	if rec == nil {
		return
	}
	key := containerNetworkKey(rec.App, rec.ID)
	netMu.Lock()
	s := loadNetState()
	netRec, ok := s.Containers[key]
	if ok {
		if key == strings.TrimSpace(rec.ID) {
			delete(s.Containers, key)
		} else {
			netRec.VethHost = ""
			netRec.VethPeer = ""
		}
		_ = saveNetState(s)
	}
	netMu.Unlock()
	if ok && netRec.VethHost != "" {
		_ = ipRun("link", "del", netRec.VethHost)
	}
	// Remove the port-forward rules installed at spawn time.
	removePortForward(rec.Port, rec.IP)
}

// ─── bridge setup ─────────────────────────────────────────────────────────────

// ensureBridge creates the station0 Linux bridge, assigns 10.88.0.1/16,
// enables IP forwarding, and adds an iptables MASQUERADE rule for outbound
// internet access from containers. Idempotent — safe to call on every run.
func ensureBridge() error {
	if err := exec.Command("ip", "link", "show", "dev", bridgeName).Run(); err == nil {
		return nil // already exists
	}
	if err := ipRun("link", "add", bridgeName, "type", "bridge"); err != nil {
		return fmt.Errorf("create bridge %s: %w", bridgeName, err)
	}
	if err := ipRun("addr", "add", bridgeIP+"/16", "dev", bridgeName); err != nil {
		if !strings.Contains(err.Error(), "File exists") {
			return fmt.Errorf("assign bridge IP: %w", err)
		}
	}
	if err := ipRun("link", "set", bridgeName, "up"); err != nil {
		return fmt.Errorf("bring up bridge: %w", err)
	}
	// Enable IP forwarding so containers can route traffic through the host.
	_ = os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1\n"), 0644)
	// Add MASQUERADE so containers reach the internet. Check first to avoid duplicates.
	if iptablesRun("-t", "nat", "-C", "POSTROUTING", "-s", bridgeSubnet, "-j", "MASQUERADE") != nil {
		_ = iptablesRun("-t", "nat", "-A", "POSTROUTING", "-s", bridgeSubnet, "-j", "MASQUERADE")
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func ipRun(args ...string) error {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func iptablesRun(args ...string) error {
	out, err := exec.Command("iptables", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ─── port forwarding ──────────────────────────────────────────────────────────

// addPortForward installs two iptables rules so host processes (e.g. the vessel
// proxy daemon) can reach a bridge-networked container at 127.0.0.1:port:
//
//   - OUTPUT DNAT:       127.0.0.1:port → containerIP:port  (host-originated traffic)
//   - POSTROUTING MASQ:  rewrite source so the container can route its reply back
//
// route_localnet must be enabled for the kernel to accept DNAT on the loopback
// interface; we set it unconditionally — it is safe and idempotent.
func addPortForward(port int, containerIP string) error {
	if port == 0 || containerIP == "" {
		return nil
	}
	_ = os.WriteFile("/proc/sys/net/ipv4/conf/all/route_localnet", []byte("1\n"), 0644)
	dport := fmt.Sprintf("%d", port)
	target := containerIP + ":" + dport
	if iptablesRun("-t", "nat", "-C", "OUTPUT", "-p", "tcp",
		"-d", "127.0.0.1", "--dport", dport, "-j", "DNAT", "--to-destination", target) != nil {
		if err := iptablesRun("-t", "nat", "-A", "OUTPUT", "-p", "tcp",
			"-d", "127.0.0.1", "--dport", dport, "-j", "DNAT", "--to-destination", target); err != nil {
			return fmt.Errorf("port-forward OUTPUT DNAT: %w", err)
		}
	}
	// MASQUERADE rewrites the source IP to station0's address so the container
	// sends its reply to the bridge gateway, which routes it back to the caller.
	if iptablesRun("-t", "nat", "-C", "POSTROUTING", "-p", "tcp",
		"-d", containerIP, "--dport", dport, "-j", "MASQUERADE") != nil {
		_ = iptablesRun("-t", "nat", "-A", "POSTROUTING", "-p", "tcp",
			"-d", containerIP, "--dport", dport, "-j", "MASQUERADE")
	}
	return nil
}

// removePortForward deletes the rules installed by addPortForward. Safe to call
// even if the rules no longer exist (iptables -D is a no-op on missing rules).
func removePortForward(port int, containerIP string) {
	if port == 0 || containerIP == "" {
		return
	}
	dport := fmt.Sprintf("%d", port)
	target := containerIP + ":" + dport
	_ = iptablesRun("-t", "nat", "-D", "OUTPUT", "-p", "tcp",
		"-d", "127.0.0.1", "--dport", dport, "-j", "DNAT", "--to-destination", target)
	_ = iptablesRun("-t", "nat", "-D", "POSTROUTING", "-p", "tcp",
		"-d", containerIP, "--dport", dport, "-j", "MASQUERADE")
}

// netAllocCmd is the handler for the hidden "_net-alloc" command.
// It allocates (or reuses) a bridge IP for the given container ID and writes
// the bare IP string to stdout — no trailing newline, so callers can trim.
func netAllocCmd(key string) {
	ip, err := allocOrReuseContainerIP(strings.TrimSpace(key))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(ip)
}

