package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/allocator"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/config"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/connector"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/registry"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/scanner"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/transport"
)

var version = "0.1.0-dev"

type EnvManager struct {
	cfg      *config.Config
	scanner  *scanner.Scanner
	registry *registry.Registry
	execPool map[string]connector.Executor
}

func main() {
	var (
		configPath = flag.String("config", "", "Path to config file")
		cmd        = flag.String("cmd", "serve", "Command: serve, scan, version")
		nodeFlag   = flag.String("node", "", "Node name for scan command")
		scopeFlag  = flag.String("scope", "all", "Scan scope: all, bare, container, ports, resources")
	)
	flag.Parse()

	switch *cmd {
	case "version":
		fmt.Println("env-manager", version)
		os.Exit(0)
	case "serve":
		if err := runServe(*configPath); err != nil {
			log.Fatal(err)
		}
	case "scan":
		if *nodeFlag == "" {
			log.Fatal("-node flag required for scan command")
		}
		if err := runScan(*configPath, *nodeFlag, *scopeFlag); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command: %s", *cmd)
	}
}

func runServe(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize executors pool
	execPool := make(map[string]connector.Executor)
	for _, node := range cfg.Nodes {
		exec, err := connector.NewExecutor(node)
		if err != nil {
			log.Printf("Failed to connect to node %s: %v", node.Name, err)
			continue
		}
		execPool[node.Name] = exec
		defer exec.Close()
	}

	// Initialize registry
	reg, err := registry.NewRegistry(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("init registry: %w", err)
	}
	defer reg.Close()

	// Initialize scanner
	s := scanner.New(execPool, cfg.Settings.ScanTimeout)

	em := &EnvManager{
		cfg:      cfg,
		scanner:  s,
		registry: reg,
		execPool: execPool,
	}

	// Setup JSON-RPC handler
	handler := transport.NewHandler(em.handleToolCall)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	log.Println("env-manager", version, "started")
	return handler.Serve(ctx, os.Stdin, os.Stdout)
}

func runScan(configPath, nodeName, scope string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var targetNode config.NodeConfig
	for _, n := range cfg.Nodes {
		if n.Name == nodeName {
			targetNode = n
			break
		}
	}
	if targetNode.Name == "" {
		return fmt.Errorf("node %s not found in config", nodeName)
	}

	exec, err := connector.NewExecutor(targetNode)
	if err != nil {
		return fmt.Errorf("connect to node: %w", err)
	}
	defer exec.Close()

	s := scanner.New(map[string]connector.Executor{nodeName: exec}, cfg.Settings.ScanTimeout)
	result, err := s.ScanNode(context.Background(), nodeName, scope)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func (em *EnvManager) handleToolCall(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "scan_node":
		var req struct {
			Node  string `json:"node"`
			Scope string `json:"scope"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		if req.Scope == "" {
			req.Scope = "all"
		}
		return em.scanner.ScanNode(ctx, req.Node, req.Scope)

	case "scan_all":
		return em.scanner.ScanAll(ctx)

	case "list_services":
		var req struct {
			Node string `json:"node"`
			Port int    `json:"port"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		return em.registry.ListServices(req.Node, req.Port, req.Name)

	case "find_port":
		var req struct {
			Port int    `json:"port"`
			Node string `json:"node"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		return em.registry.FindPort(req.Port, req.Node)

	case "preflight_check":
		var req struct {
			Node         string                `json:"node"`
			Requirements scanner.Requirements  `json:"requirements"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		pf := allocator.NewPreflight(em.scanner, em.registry)
		return pf.Check(ctx, req.Node, req.Requirements)

	case "allocate_port":
		var req struct {
			Node        string `json:"node"`
			Preferred   int    `json:"preferred"`
			RangeStart  int    `json:"range_start"`
			RangeEnd    int    `json:"range_end"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, err
		}
		alloc := allocator.NewPortAllocator(em.registry)
		return alloc.Allocate(ctx, req.Node, req.Preferred, req.RangeStart, req.RangeEnd)

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}
