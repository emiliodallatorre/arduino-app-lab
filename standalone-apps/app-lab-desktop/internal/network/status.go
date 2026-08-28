package network

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/arduino/arduino-app-cli/pkg/board/remote"
)

const connectivityCheckURL = "http://connectivitycheck.gstatic.com/generate_204"

type Status string

var (
	ConnectedStatus    Status = "connected"
	ConnectingStatus   Status = "connecting"
	DisconnectedStatus Status = "disconnected"
)

func (m *Manager) GetStatusByType(ctx context.Context, netType string) (Status, error) {
	// -t = terse, -f = columns  TYPE,STATE
	out, err := m.Run(ctx, "-t", "-f", "TYPE,STATE", "device")
	if err != nil {
		return DisconnectedStatus, fmt.Errorf("failed to query devices: %w", err)
	}

	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == netType && strings.TrimSpace(parts[1]) == "connected" {
			return ConnectedStatus, nil
		}

		if parts[0] == netType && strings.TrimSpace(parts[1]) == "connecting" {
			return ConnectingStatus, nil
		}
	}
	return DisconnectedStatus, nil
}

func (nm *Manager) getInternetStatus(ctx context.Context) (bool, error) {
	cmd := nm.Conn.GetCmd(
		"curl",
		"--silent",
		"--output", "/dev/null",
		"--write-out", "%{http_code}",
		"--connect-timeout", "2",
		"--max-time", "5",
		connectivityCheckURL,
	)
	probeCtx, cancel := context.WithTimeout(ctx, nm.Timeout)
	defer cancel()

	statusCode, probeErr := cmd.Output(probeCtx)
	if probeErr == nil {
		httpStatus := strings.TrimSpace(string(statusCode))
		connected := httpStatus == "204"
		slog.Info(
			"Internet connectivity HTTP probe completed",
			"url", connectivityCheckURL,
			"statusCode", httpStatus,
			"connected", connected,
		)
		if connected {
			return true, nil
		}
	} else {
		slog.Info("Internet connectivity HTTP probe failed", "url", connectivityCheckURL, "error", probeErr)
	}

	out, err := nm.Run(ctx, "networking", "connectivity", "check")
	if err != nil {
		slog.Info("NetworkManager connectivity check failed", "error", err)
		return false, nil
	}

	nmStatus := strings.TrimSpace(out)
	connected := nmStatus == "full"
	slog.Info("NetworkManager connectivity check completed", "status", nmStatus, "connected", connected)
	return connected, nil
}

func GetInternetStatus(ctx context.Context, conn remote.RemoteConn) (bool, error) {
	if conn == nil {
		return false, fmt.Errorf("missing connection")
	}
	nm := &Manager{
		Timeout: 5 * time.Second,
		Conn:    conn,
	}
	return nm.getInternetStatus(ctx)
}

func GetConnectionName(ctx context.Context, conn remote.RemoteConn) (*string, error) {
	if conn == nil {
		return nil, fmt.Errorf("missing connection")
	}
	nm := &Manager{
		Timeout: 5 * time.Second,
		Conn:    conn,
	}
	// -t = terse, -f = columns NAME
	out, err := nm.Run(ctx, "-t", "-f", "NAME", "connection", "show", "--active")
	if err != nil {
		return nil, fmt.Errorf("failed to query connection: %w", err)
	}
	parts := strings.Split(out, "\n")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return nil, nil
	}
	name := strings.TrimSpace(parts[0])
	return &name, nil
}

func (nm *Manager) getIPAddress(ctx context.Context) (*string, error) {
	out, err := nm.Run(ctx, "-g", "GENERAL.TYPE,IP4.ADDRESS", "device", "show")
	if err != nil {
		return nil, fmt.Errorf("failed to query ip address: %w", err)
	}

	type networkIP struct {
		Type string
		IP   string
	}
	var networksIPs []networkIP
	for _, group := range strings.Split(out, "\n\n") {
		parts := strings.Split(group, "\n")
		if len(parts) < 2 {
			continue
		}
		netType := strings.TrimSpace(parts[0])

		ip := strings.TrimSpace(parts[1])
		parts = strings.Split(ip, "/")
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			continue
		}
		ip = strings.TrimSpace(parts[0])

		networksIPs = append(networksIPs, networkIP{
			Type: netType,
			IP:   ip,
		})
	}

	if idx := slices.IndexFunc(networksIPs, func(n networkIP) bool {
		return n.Type == "ethernet"
	}); idx != -1 {
		return &networksIPs[idx].IP, nil
	}

	if idx := slices.IndexFunc(networksIPs, func(n networkIP) bool {
		return n.Type == "wifi"
	}); idx != -1 {
		return &networksIPs[idx].IP, nil
	}

	return nil, fmt.Errorf("no IP address found for ethernet or wifi connections")
}

func GetIPAddress(ctx context.Context, conn remote.RemoteConn) (*string, error) {
	if conn == nil {
		return nil, fmt.Errorf("missing connection")
	}
	nm := &Manager{
		Timeout: 5 * time.Second,
		Conn:    conn,
	}
	return nm.getIPAddress(ctx)
}
