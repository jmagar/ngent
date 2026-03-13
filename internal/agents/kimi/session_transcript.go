package kimi

import (
	"context"
	"errors"
	"fmt"

	"github.com/beyond5959/ngent/internal/agents"
	"github.com/beyond5959/ngent/internal/agents/acpsession"
	"github.com/beyond5959/ngent/internal/agents/acpstdio"
)

var _ agents.SessionTranscriptLoader = (*Client)(nil)

// LoadSessionTranscript replays one Kimi session through ACP session/load.
func (c *Client) LoadSessionTranscript(
	ctx context.Context,
	req agents.SessionTranscriptRequest,
) (agents.SessionTranscriptResult, error) {
	if c == nil {
		return agents.SessionTranscriptResult{}, errors.New("kimi: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	session, err := agents.FindSessionByID(ctx, c, req.CWD, req.SessionID)
	if err != nil {
		return agents.SessionTranscriptResult{}, err
	}

	conn, cleanup, _, initResult, err := c.openConn(ctx, c.CurrentModelID(), c.CurrentConfigOverrides())
	if err != nil {
		return agents.SessionTranscriptResult{}, err
	}
	defer cleanup()

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

	if _, err := conn.Call(ctx, "session/load", kimiSessionLoadParams(c, session.SessionID)); err != nil {
		return agents.SessionTranscriptResult{}, fmt.Errorf("kimi: session/load: %w", err)
	}
	return collector.Result(), nil
}
