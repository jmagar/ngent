package kimi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/beyond5959/ngent/internal/agents"
	"github.com/beyond5959/ngent/internal/agents/acpmodel"
	"github.com/beyond5959/ngent/internal/agents/acpstdio"
	"github.com/beyond5959/ngent/internal/agents/agentutil"
)

const (
	updateTypeMessageChunk       = "agent_message_chunk"
	methodSessionSetConfigOption = "session/set_config_option"

	defaultPermissionTimeout = 15 * time.Second
)

// Config configures the Kimi CLI ACP stdio provider.
type Config = agentutil.Config

// Client runs one Kimi ACP process per Stream call.
type Client struct {
	*agentutil.State
}

type commandSpec struct {
	mode  string
	label string
}

var _ agents.Streamer = (*Client)(nil)
var _ agents.ConfigOptionManager = (*Client)(nil)

// New constructs a Kimi ACP client.
func New(cfg Config) (*Client, error) {
	state, err := agentutil.NewState("kimi", cfg)
	if err != nil {
		return nil, err
	}
	return &Client{
		State: state,
	}, nil
}

// Preflight checks that the kimi binary is available in PATH.
func Preflight() error {
	return agentutil.PreflightBinary("kimi")
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "kimi" }

// ConfigOptions queries ACP session config options.
func (c *Client) ConfigOptions(ctx context.Context) ([]agents.ConfigOption, error) {
	if c == nil {
		return nil, errors.New("kimi: nil client")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return c.runConfigSession(ctx, c.CurrentModelID(), c.CurrentConfigOverrides(), "", "")
}

// SetConfigOption applies one ACP session config option.
func (c *Client) SetConfigOption(ctx context.Context, configID, value string) ([]agents.ConfigOption, error) {
	if c == nil {
		return nil, errors.New("kimi: nil client")
	}
	configID = strings.TrimSpace(configID)
	value = strings.TrimSpace(value)
	if configID == "" {
		return nil, errors.New("kimi: configID is required")
	}
	if value == "" {
		return nil, errors.New("kimi: value is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	options, err := c.runConfigSession(ctx, c.CurrentModelID(), c.CurrentConfigOverrides(), configID, value)
	if err != nil {
		return nil, err
	}
	c.ApplyConfigOptionResult(configID, value, options)
	return options, nil
}

// Stream spawns Kimi in ACP mode, runs one turn, and streams deltas via onDelta.
func (c *Client) Stream(ctx context.Context, input string, onDelta func(delta string) error) (agents.StopReason, error) {
	if c == nil {
		return agents.StopReasonEndTurn, errors.New("kimi: nil client")
	}
	if onDelta == nil {
		return agents.StopReasonEndTurn, errors.New("kimi: onDelta callback is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	modelID := c.CurrentModelID()
	configOverrides := c.CurrentConfigOverrides()

	conn, cleanup, _, err := c.openConn(ctx, modelID)
	if err != nil {
		return agents.StopReasonEndTurn, err
	}
	defer cleanup()

	newResult, err := conn.Call(ctx, "session/new", sessionNewParams(c.Dir(), modelID))
	if err != nil {
		return agents.StopReasonEndTurn, fmt.Errorf("kimi: session/new: %w", err)
	}
	sessionID := acpstdio.ParseSessionID(newResult)
	if sessionID == "" {
		return agents.StopReasonEndTurn, errors.New("kimi: session/new returned empty sessionId")
	}
	if _, err := c.applyConfigOverrides(ctx, conn, sessionID, acpmodel.ExtractConfigOptions(newResult), configOverrides); err != nil {
		return agents.StopReasonEndTurn, err
	}

	permHandler, hasPermHandler := agents.PermissionHandlerFromContext(ctx)
	conn.SetRequestHandler(func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method != "session/request_permission" {
			return nil, &acpstdio.RPCError{Code: acpstdio.MethodNotFound, Message: "method not found"}
		}

		var req struct {
			SessionID string `json:"sessionId"`
			ToolCall  struct {
				Title string `json:"title"`
				Kind  string `json:"kind"`
			} `json:"toolCall"`
			Options []struct {
				OptionID string `json:"optionId"`
				Kind     string `json:"kind"`
			} `json:"options"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return buildDeclinedPermissionResponse(req.Options)
		}

		if !hasPermHandler {
			return buildDeclinedPermissionResponse(req.Options)
		}

		permCtx, cancel := context.WithTimeout(ctx, defaultPermissionTimeout)
		defer cancel()

		resp, err := permHandler(permCtx, agents.PermissionRequest{
			Approval:  strings.TrimSpace(req.ToolCall.Title),
			Command:   strings.TrimSpace(req.ToolCall.Kind),
			RawParams: map[string]any{"sessionId": req.SessionID},
		})
		if err != nil {
			return buildDeclinedPermissionResponse(req.Options)
		}

		switch resp.Outcome {
		case agents.PermissionOutcomeApproved:
			return buildApprovedPermissionResponse(req.Options)
		case agents.PermissionOutcomeCancelled:
			return buildCancelledPermissionResponse()
		default:
			return buildDeclinedPermissionResponse(req.Options)
		}
	})

	conn.SetNotificationHandler(func(msg acpstdio.Message) error {
		if msg.Method != "session/update" || len(msg.Params) == 0 {
			return nil
		}
		var payload struct {
			Update struct {
				SessionUpdate string `json:"sessionUpdate"`
				Content       struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"update"`
		}
		if err := json.Unmarshal(msg.Params, &payload); err != nil {
			return nil
		}
		if payload.Update.SessionUpdate != updateTypeMessageChunk {
			return nil
		}
		text := payload.Update.Content.Text
		if text == "" {
			return nil
		}
		return onDelta(text)
	})

	stopCancelWatch := make(chan struct{})
	defer close(stopCancelWatch)
	go func() {
		select {
		case <-ctx.Done():
			c.sendCancel(conn, sessionID)
		case <-stopCancelWatch:
		}
	}()

	promptParams := map[string]any{
		"sessionId": sessionID,
		"prompt":    []map[string]any{{"type": "text", "text": input}},
	}
	if modelID != "" {
		promptParams["model"] = modelID
	}

	promptResult, err := conn.Call(ctx, "session/prompt", promptParams)
	if err != nil {
		if ctx.Err() != nil {
			return agents.StopReasonCancelled, nil
		}
		return agents.StopReasonEndTurn, fmt.Errorf("kimi: session/prompt: %w", err)
	}

	if acpstdio.ParseStopReason(promptResult) == "cancelled" {
		return agents.StopReasonCancelled, nil
	}
	return agents.StopReasonEndTurn, nil
}

func (c *Client) sendCancel(conn *acpstdio.Conn, sessionID string) {
	conn.Notify("session/cancel", map[string]any{"sessionId": sessionID})
}

func buildApprovedPermissionResponse(options []struct {
	OptionID string `json:"optionId"`
	Kind     string `json:"kind"`
}) (json.RawMessage, error) {
	optionID := pickPermissionOptionID(options, "allow_once", "allow_always")
	if optionID == "" {
		return buildDeclinedPermissionResponse(options)
	}
	return buildSelectedPermissionResponse(optionID)
}

func buildDeclinedPermissionResponse(options []struct {
	OptionID string `json:"optionId"`
	Kind     string `json:"kind"`
}) (json.RawMessage, error) {
	optionID := pickPermissionOptionID(options, "reject_once", "reject_always")
	if optionID == "" {
		return buildCancelledPermissionResponse()
	}
	return buildSelectedPermissionResponse(optionID)
}

func buildSelectedPermissionResponse(optionID string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"outcome": map[string]any{
			"outcome":  "selected",
			"optionId": optionID,
		},
	})
}

func buildCancelledPermissionResponse() (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"outcome": map[string]any{
			"outcome": "cancelled",
		},
	})
}

func pickPermissionOptionID(options []struct {
	OptionID string `json:"optionId"`
	Kind     string `json:"kind"`
}, preferredKinds ...string) string {
	for _, kind := range preferredKinds {
		for _, option := range options {
			if strings.TrimSpace(option.OptionID) == "" {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(option.Kind), kind) {
				return strings.TrimSpace(option.OptionID)
			}
		}
	}
	return ""
}

func (c *Client) runConfigSession(
	ctx context.Context,
	modelID string,
	configOverrides map[string]string,
	configID, value string,
) ([]agents.ConfigOption, error) {
	sessionModelID := modelID
	if strings.EqualFold(configID, "model") && value != "" {
		sessionModelID = value
	}

	conn, cleanup, _, err := c.openConn(ctx, sessionModelID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	newResult, err := conn.Call(ctx, "session/new", sessionNewParams(c.Dir(), sessionModelID))
	if err != nil {
		return nil, fmt.Errorf("kimi: config options session/new: %w", err)
	}
	sessionID := acpstdio.ParseSessionID(newResult)
	if sessionID == "" {
		return nil, errors.New("kimi: config options session/new returned empty sessionId")
	}

	options, err := c.applyConfigOverrides(ctx, conn, sessionID, acpmodel.ExtractConfigOptions(newResult), configOverrides)
	if err != nil {
		return nil, err
	}
	if configID == "" {
		return options, nil
	}
	if strings.EqualFold(configID, "model") {
		return options, nil
	}
	setResult, err := conn.Call(ctx, methodSessionSetConfigOption, map[string]any{
		"sessionId": sessionID,
		"configId":  configID,
		"value":     value,
	})
	if err != nil {
		return nil, fmt.Errorf("kimi: config options session/set_config_option: %w", err)
	}

	updated := acpmodel.ExtractConfigOptions(setResult)
	if len(updated) == 0 {
		return options, nil
	}
	return updated, nil
}

func (c *Client) applyConfigOverrides(
	ctx context.Context,
	conn *acpstdio.Conn,
	sessionID string,
	options []agents.ConfigOption,
	overrides map[string]string,
) ([]agents.ConfigOption, error) {
	if len(overrides) == 0 {
		return options, nil
	}

	configIDs := make([]string, 0, len(overrides))
	for configID := range overrides {
		configIDs = append(configIDs, configID)
	}
	sort.Strings(configIDs)

	current := options
	for _, configID := range configIDs {
		value := strings.TrimSpace(overrides[configID])
		if value == "" {
			continue
		}
		setResult, err := conn.Call(ctx, methodSessionSetConfigOption, map[string]any{
			"sessionId": sessionID,
			"configId":  configID,
			"value":     value,
		})
		if err != nil {
			return nil, fmt.Errorf("kimi: session/set_config_option(%s): %w", configID, err)
		}
		if updated := acpmodel.ExtractConfigOptions(setResult); len(updated) > 0 {
			current = updated
		}
	}
	return current, nil
}

func (c *Client) openConn(ctx context.Context, modelID string) (*acpstdio.Conn, func(), string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	specs := commandCandidates()
	var attemptErrors []string
	for idx, spec := range specs {
		conn, cleanup, err := c.openConnWithCommand(ctx, spec, modelID)
		if err == nil {
			return conn, cleanup, spec.label, nil
		}
		attemptErrors = append(attemptErrors, err.Error())
		if idx == len(specs)-1 || !shouldRetryACPStartup(err) {
			break
		}
	}

	return nil, nil, "", fmt.Errorf(
		"kimi: failed to start ACP mode (%s)",
		strings.Join(attemptErrors, "; "),
	)
}

func (c *Client) openConnWithCommand(ctx context.Context, spec commandSpec, modelID string) (*acpstdio.Conn, func(), error) {
	cmd := exec.Command("kimi", spec.args(modelID)...)
	cmd.Dir = c.Dir()
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("kimi: %s open stdin pipe: %w", spec.label, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("kimi: %s open stdout pipe: %w", spec.label, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("kimi: %s open stderr pipe: %w", spec.label, err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("kimi: %s start process: %w", spec.label, err)
	}

	errCh := make(chan error, 1)
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go func() { errCh <- cmd.Wait() }()

	conn := acpstdio.NewConn(stdin, stdout, "kimi")
	cleanup := func() {
		conn.Close()
		acpstdio.TerminateProcess(cmd, errCh, 2*time.Second)
	}

	if _, err := conn.Call(ctx, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs": map[string]any{
				"readTextFile":  false,
				"writeTextFile": false,
			},
		},
	}); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("kimi: %s initialize: %w", spec.label, err)
	}

	return conn, cleanup, nil
}

func commandCandidates() []commandSpec {
	// Official Kimi docs currently show both `kimi acp` and `kimi --acp`.
	return []commandSpec{
		{mode: "subcommand", label: "kimi acp"},
		{mode: "flag", label: "kimi --acp"},
	}
}

func (s commandSpec) args(modelID string) []string {
	args := make([]string, 0, 3)
	if strings.TrimSpace(modelID) != "" {
		args = append(args, "--model", strings.TrimSpace(modelID))
	}
	switch s.mode {
	case "flag":
		args = append(args, "--acp")
	default:
		args = append(args, "acp")
	}
	return args
}

func sessionNewParams(dir, modelID string) map[string]any {
	params := map[string]any{
		"cwd":        dir,
		"mcpServers": []any{},
	}
	modelID = strings.TrimSpace(modelID)
	if modelID != "" {
		// Kimi's ACP runtime may derive model selection from process startup
		// flags, but sending both fields keeps handshake responses aligned when
		// the server also honors session/new model hints.
		params["model"] = modelID
		params["modelId"] = modelID
	}
	return params
}

func shouldRetryACPStartup(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection closed") ||
		strings.Contains(message, "start process") ||
		strings.Contains(message, "initialize")
}
