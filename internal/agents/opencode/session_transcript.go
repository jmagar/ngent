package opencode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/beyond5959/ngent/internal/agents"
	"github.com/beyond5959/ngent/internal/agents/acpsession"
	"github.com/beyond5959/ngent/internal/agents/acpstdio"
)

var _ agents.SessionTranscriptLoader = (*Client)(nil)

// LoadSessionTranscript replays one OpenCode session through ACP session/load.
func (c *Client) LoadSessionTranscript(
	ctx context.Context,
	req agents.SessionTranscriptRequest,
) (agents.SessionTranscriptResult, error) {
	if c == nil {
		return agents.SessionTranscriptResult{}, errors.New("opencode: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	session, err := agents.FindSessionByID(ctx, c, req.CWD, req.SessionID)
	if err != nil {
		return agents.SessionTranscriptResult{}, err
	}

	cmd := exec.Command("opencode", "acp", "--cwd", c.Dir())
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: transcript open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: transcript open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: transcript open stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: transcript start process: %w", err)
	}

	errCh := make(chan error, 1)
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go func() { errCh <- cmd.Wait() }()

	conn := acpstdio.NewConn(stdin, stdout, "opencode")
	defer conn.Close()
	defer acpstdio.TerminateProcess(cmd, errCh, 2*time.Second)

	initResult, err := conn.Call(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "ngent",
			"version": "0.1.0",
		},
		"protocolVersion": 1,
	})
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: transcript initialize: %w", err)
	}

	caps := acpsession.ParseInitializeCapabilities(initResult)
	if !caps.CanLoad {
		return agents.SessionTranscriptResult{}, agents.ErrSessionLoadUnsupported
	}

	collector := agents.NewACPTranscriptCollector()
	conn.SetNotificationHandler(func(msg acpstdio.Message) error {
		if msg.Method != "session/update" || len(msg.Params) == 0 {
			return nil
		}
		return collector.HandleRawUpdate(msg.Params)
	})

	if _, err := conn.Call(ctx, "session/load", opencodeSessionLoadParams(c, session.SessionID)); err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("opencode: session/load: %w", err)
	}
	return collector.Result(), nil
}
