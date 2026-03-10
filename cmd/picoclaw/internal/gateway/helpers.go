package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/channels"
	_ "github.com/sipeed/picoclaw/pkg/channels/dingtalk"
	_ "github.com/sipeed/picoclaw/pkg/channels/discord"
	_ "github.com/sipeed/picoclaw/pkg/channels/feishu"
	_ "github.com/sipeed/picoclaw/pkg/channels/irc"
	_ "github.com/sipeed/picoclaw/pkg/channels/line"
	_ "github.com/sipeed/picoclaw/pkg/channels/maixcam"
	_ "github.com/sipeed/picoclaw/pkg/channels/matrix"
	_ "github.com/sipeed/picoclaw/pkg/channels/onebot"
	_ "github.com/sipeed/picoclaw/pkg/channels/pico"
	_ "github.com/sipeed/picoclaw/pkg/channels/qq"
	_ "github.com/sipeed/picoclaw/pkg/channels/slack"
	_ "github.com/sipeed/picoclaw/pkg/channels/telegram"
	_ "github.com/sipeed/picoclaw/pkg/channels/wecom"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp"
	_ "github.com/sipeed/picoclaw/pkg/channels/whatsapp_native"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/sipeed/picoclaw/pkg/devices"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/heartbeat"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/state"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/pkg/voice"
	"github.com/sipeed/picoclaw/web/whiteboard"
)

type pinchtabOptions struct {
	Enabled  bool
	ExecPath string
	Args     []string
	Env      []string
	Health   string
}

func envBool(name string) bool {
	v := strings.TrimSpace(os.Getenv(name))
	v = strings.ToLower(v)
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func pinchtabFromEnv() pinchtabOptions {
	// Opt-in only: default disabled so we don't surprise existing users.
	// Enable via: PICOCLAW_PINCHTAB_AUTOSPAWN=1
	enabled := envBool("PICOCLAW_PINCHTAB_AUTOSPAWN")

	execPath := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_EXEC"))
	if execPath == "" {
		execPath = "pinchtab"
	}

	// Default to dashboard server mode; it provides /health and can proxy to a launched bridge.
	args := []string{"server"}
	if raw := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_ARGS")); raw != "" {
		args = strings.Fields(raw)
	}

	bind := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_BIND"))
	if bind == "" {
		bind = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_PORT"))
	if port == "" {
		port = "9870"
	}

	healthURL := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_HEALTH"))
	if healthURL == "" {
		healthURL = fmt.Sprintf("http://%s/health", bind+":"+port)
	}

	// Pass through PinchTab's own env vars while allowing PicoClaw-specific overrides.
	// These are intentionally minimal so we don't hard-code behavior.
	env := []string{
		"PINCHTAB_BIND=" + bind,
		"PINCHTAB_PORT=" + port,
	}
	if v := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_STRATEGY")); v != "" {
		env = append(env, "PINCHTAB_STRATEGY="+v)
	}
	if v := strings.TrimSpace(os.Getenv("PICOCLAW_PINCHTAB_PROFILES_DIR")); v != "" {
		env = append(env, "PINCHTAB_PROFILES_DIR="+v)
	}

	return pinchtabOptions{
		Enabled:  enabled,
		ExecPath: execPath,
		Args:     args,
		Env:      env,
		Health:   healthURL,
	}
}

func scanLines(r io.Reader, onLine func(string)) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for s.Scan() {
		onLine(s.Text())
	}
}

func startPinchTab(ctx context.Context, opt pinchtabOptions) (*exec.Cmd, error) {
	if !opt.Enabled {
		return nil, nil
	}

	cmd := exec.CommandContext(ctx, opt.ExecPath, opt.Args...)
	cmd.Env = append(os.Environ(), opt.Env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	logger.InfoCF("pinchtab", "PinchTab started", map[string]any{"pid": cmd.Process.Pid, "health": opt.Health})

	go scanLines(stdout, func(line string) {
		logger.InfoCF("pinchtab", line, nil)
	})
	go scanLines(stderr, func(line string) {
		logger.WarnCF("pinchtab", line, nil)
	})

	// Probe health in background (best-effort) so users have a clear signal.
	go func() {
		client := http.Client{Timeout: 1 * time.Second}
		for i := 0; i < 40; i++ {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(250 * time.Millisecond)
			resp, err := client.Get(opt.Health)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					logger.InfoCF("pinchtab", "PinchTab health OK", nil)
					return
				}
			}
		}
		logger.WarnCF("pinchtab", "PinchTab health probe timed out", map[string]any{"health": opt.Health})
	}()

	return cmd, nil
}

func stopPinchTab(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		logger.InfoCF("pinchtab", "PinchTab stopped (windows kill)", map[string]any{"pid": pid})
		return
	}
	// Best-effort graceful stop.
	_ = cmd.Process.Signal(syscall.SIGTERM)
	logger.InfoCF("pinchtab", "PinchTab stop signal sent", map[string]any{"pid": pid})
}

func gatewayCmd(debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	// Log to file as well
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// Setup cron tool and service
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	cronService := setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)
	heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		var response string
		response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		// For heartbeat, always return silent - the subagent result will be
		// sent to user via processSystemMessage when the async task completes
		return tools.SilentResult(response)
	})

	// Create media store for file lifecycle management with TTL cleanup
	mediaStore := media.NewFileMediaStoreWithCleanup(media.MediaCleanerConfig{
		Enabled:  cfg.Tools.MediaCleanup.Enabled,
		MaxAge:   time.Duration(cfg.Tools.MediaCleanup.MaxAge) * time.Minute,
		Interval: time.Duration(cfg.Tools.MediaCleanup.Interval) * time.Minute,
	})
	mediaStore.Start()

	channelManager, err := channels.NewManager(cfg, msgBus, mediaStore)
	if err != nil {
		mediaStore.Stop()
		return fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager and media store into agent loop
	agentLoop.SetChannelManager(channelManager)
	agentLoop.SetMediaStore(mediaStore)

	// Wire up voice transcription if a supported provider is configured.
	if transcriber := voice.DetectTranscriber(cfg); transcriber != nil {
		agentLoop.SetTranscriber(transcriber)
		logger.InfoCF("voice", "Transcription enabled (agent-level)", map[string]any{"provider": transcriber.Name()})
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Optional PinchTab sidecar (opt-in via env: PICOCLAW_PINCHTAB_AUTOSPAWN=1)
	pinchOpt := pinchtabFromEnv()
	pinchCmd, pinchErr := startPinchTab(ctx, pinchOpt)
	if pinchErr != nil {
		logger.WarnCF("pinchtab", "Failed to start PinchTab", map[string]any{"error": pinchErr.Error()})
	}

	if err := cronService.Start(); err != nil {
		fmt.Printf("Error starting cron service: %v\n", err)
	}
	fmt.Println("✓ Cron service started")

	if err := heartbeatService.Start(); err != nil {
		fmt.Printf("Error starting heartbeat service: %v\n", err)
	}
	fmt.Println("✓ Heartbeat service started")

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	// Setup shared HTTP server with health endpoints and webhook handlers
	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)
	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	channelManager.SetupHTTPServer(addr, healthServer)
	
	// Register whiteboard routes on the shared mux
	channelManager.Mux().Handle("/whiteboard/", http.StripPrefix("/whiteboard", whiteboard.Handler()))

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
		return err
	}

	fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	if cp, ok := provider.(providers.StatefulProvider); ok {
		cp.Close()
	}
	cancel()
	msgBus.Close()
	stopPinchTab(pinchCmd)

	// Use a fresh context with timeout for graceful shutdown,
	// since the original ctx is already canceled.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	channelManager.StopAll(shutdownCtx)
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	mediaStore.Stop()
	agentLoop.Stop()
	fmt.Println("✓ Gateway stopped")

	return nil
}

func setupCronTool(
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
	workspace string,
	restrict bool,
	execTimeout time.Duration,
	cfg *config.Config,
) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool if enabled
	var cronTool *tools.CronTool
	if cfg.Tools.IsToolEnabled("cron") {
		var err error
		cronTool, err = tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
		if err != nil {
			log.Fatalf("Critical error during CronTool initialization: %v", err)
		}

		agentLoop.RegisterTool(cronTool)
	}

	// Set onJob handler
	if cronTool != nil {
		cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
			result := cronTool.ExecuteJob(context.Background(), job)
			return result, nil
		})
	}

	return cronService
}
