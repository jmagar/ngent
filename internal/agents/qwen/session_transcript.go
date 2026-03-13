package qwen

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

// LoadSessionTranscript replays one Qwen session through ACP session/load.
func (c *Client) LoadSessionTranscript(
	ctx context.Context,
	req agents.SessionTranscriptRequest,
) (agents.SessionTranscriptResult, error) {
	if c == nil {
		return agents.SessionTranscriptResult{}, errors.New("qwen: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	session, err := agents.FindSessionByID(ctx, c, req.CWD, req.SessionID)
	if err != nil {
		return agents.SessionTranscriptResult{}, err
	}

	cmd := exec.Command("qwen", "--acp")
	cmd.Dir = c.Dir()
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: transcript open stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: transcript open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: transcript open stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: transcript start process: %w", err)
	}

	errCh := make(chan error, 1)
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go func() { errCh <- cmd.Wait() }()

	conn := acpstdio.NewConn(stdin, stdout, "qwen")
	defer conn.Close()
	defer acpstdio.TerminateProcess(cmd, errCh, 2*time.Second)

	initResult, err := conn.Call(ctx, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs": map[string]any{
				"readTextFile":  false,
				"writeTextFile": false,
			},
		},
	})
	if err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: transcript initialize: %w", err)
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

	if _, err := conn.Call(ctx, "session/load", qwenSessionLoadParams(c, session.SessionID)); err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("qwen: session/load: %w", err)
	}
	return collector.Result(), nil
}
