package network

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/arduino/arduino-app-cli/pkg/board/remote"
	"github.com/stretchr/testify/require"
)

type commandResponse struct {
	output string
	err    error
}

type fakeCmder struct {
	response commandResponse
}

func (c *fakeCmder) Run(context.Context) error {
	return c.response.err
}

func (c *fakeCmder) Output(context.Context) ([]byte, error) {
	return []byte(c.response.output), c.response.err
}

func (c *fakeCmder) Interactive() (io.WriteCloser, io.Reader, io.Reader, remote.Closer, error) {
	return nil, nil, nil, nil, errors.New("interactive commands are not supported")
}

type fakeRemoteConn struct {
	responses map[string]commandResponse
	commands  []string
}

func (c *fakeRemoteConn) GetCmd(name string, _ ...string) remote.Cmder {
	c.commands = append(c.commands, name)
	return &fakeCmder{response: c.responses[name]}
}

func (c *fakeRemoteConn) List(string) ([]remote.FileInfo, error) {
	return nil, nil
}

func (c *fakeRemoteConn) MkDirAll(string) error {
	return nil
}

func (c *fakeRemoteConn) WriteFile(io.Reader, string) error {
	return nil
}

func (c *fakeRemoteConn) ReadFile(string) (io.ReadCloser, error) {
	return nil, nil
}

func (c *fakeRemoteConn) Remove(string) error {
	return nil
}

func (c *fakeRemoteConn) Stats(string) (remote.FileInfo, error) {
	return remote.FileInfo{}, nil
}

func (c *fakeRemoteConn) Forward(context.Context, int, int) error {
	return nil
}

func (c *fakeRemoteConn) ForwardKillAll(context.Context) error {
	return nil
}

func (c *fakeRemoteConn) Push(context.Context, string, string) error {
	return nil
}

func TestGetInternetStatus(t *testing.T) {
	tests := []struct {
		name          string
		responses     map[string]commandResponse
		wantConnected bool
		wantNmcli     bool
	}{
		{
			name: "HTTP probe reports connectivity without using NetworkManager",
			responses: map[string]commandResponse{
				"curl": {output: "204"},
			},
			wantConnected: true,
		},
		{
			name: "NetworkManager confirms connectivity after HTTP probe is inconclusive",
			responses: map[string]commandResponse{
				"curl":  {output: "200"},
				"nmcli": {output: "full"},
			},
			wantConnected: true,
			wantNmcli:     true,
		},
		{
			name: "NetworkManager confirms connectivity after HTTP probe fails",
			responses: map[string]commandResponse{
				"curl":  {err: errors.New("request failed")},
				"nmcli": {output: "full"},
			},
			wantConnected: true,
			wantNmcli:     true,
		},
		{
			name: "NetworkManager rejects captive portal response",
			responses: map[string]commandResponse{
				"curl":  {output: "200"},
				"nmcli": {output: "limited"},
			},
			wantNmcli: true,
		},
		{
			name: "failed HTTP and NetworkManager checks report offline",
			responses: map[string]commandResponse{
				"curl":  {err: errors.New("request failed")},
				"nmcli": {err: errors.New("NetworkManager unavailable")},
			},
			wantNmcli: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &fakeRemoteConn{responses: tt.responses}
			manager := &Manager{
				Timeout: 5 * time.Second,
				Conn:    conn,
			}

			connected, err := manager.getInternetStatus(context.Background())

			require.NoError(t, err)
			require.Equal(t, tt.wantConnected, connected)
			require.Equal(t, tt.wantNmcli, slices.Contains(conn.commands, "nmcli"))
		})
	}
}
