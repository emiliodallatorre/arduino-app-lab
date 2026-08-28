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
		wantCurl      bool
	}{
		{
			name: "NetworkManager reports full connectivity",
			responses: map[string]commandResponse{
				"nmcli": {output: "full"},
			},
			wantConnected: true,
		},
		{
			name: "HTTP probe detects connectivity for unmanaged network",
			responses: map[string]commandResponse{
				"nmcli": {output: "unknown"},
				"curl":  {output: "204"},
			},
			wantConnected: true,
			wantCurl:      true,
		},
		{
			name: "HTTP probe handles NetworkManager error",
			responses: map[string]commandResponse{
				"nmcli": {err: errors.New("NetworkManager unavailable")},
				"curl":  {output: "204"},
			},
			wantConnected: true,
			wantCurl:      true,
		},
		{
			name: "HTTP probe rejects captive portal response",
			responses: map[string]commandResponse{
				"nmcli": {output: "limited"},
				"curl":  {output: "200"},
			},
			wantCurl: true,
		},
		{
			name: "HTTP probe reports failed request as offline",
			responses: map[string]commandResponse{
				"nmcli": {output: "none"},
				"curl":  {err: errors.New("request failed")},
			},
			wantCurl: true,
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
			require.Equal(t, tt.wantCurl, slices.Contains(conn.commands, "curl"))
		})
	}
}
