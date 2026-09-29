package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/delaakakpo/tunnel-inspector/internal/config"
	"github.com/delaakakpo/tunnel-inspector/internal/proxy"
	"github.com/delaakakpo/tunnel-inspector/internal/requests"
	"github.com/delaakakpo/tunnel-inspector/internal/server"
	"github.com/delaakakpo/tunnel-inspector/internal/tunnel"
	"github.com/delaakakpo/tunnel-inspector/internal/websocket"
	"github.com/delaakakpo/tunnel-inspector/web"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("tunnel-inspector v%s\n", version)
		os.Exit(0)
	}

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "start" {
		args = args[1:]
	}

	cfg := config.NewDefaultConfig()

	fs := flag.NewFlagSet("tunnel-inspector", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: tunnel-inspector start [options]")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
	}

	fs.IntVar(&cfg.AppPort, "port", cfg.AppPort, "Target application port (e.g. 8000)")
	fs.IntVar(&cfg.AppPort, "p", cfg.AppPort, "Target application port (shorthand)")
	fs.StringVar(&cfg.AppHost, "host", cfg.AppHost, "Target application host")
	fs.IntVar(&cfg.DashboardPort, "dashboard-port", cfg.DashboardPort, "Dashboard web UI port")
	fs.IntVar(&cfg.ProxyPort, "proxy-port", cfg.ProxyPort, "Local proxy port for cloudflared")
	fs.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite database storage path")
	fs.Int64Var(&cfg.MaxBodySize, "max-body", cfg.MaxBodySize, "Maximum body size to capture in bytes")
	fs.BoolVar(&cfg.NoTunnel, "no-tunnel", cfg.NoTunnel, "Run inspection proxy without starting cloudflared")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Verify or find available proxy port if default is occupied
	cfg.ProxyPort = ensurePortAvailable(cfg.ProxyHost, cfg.ProxyPort)

	// 1. Initialize SQLite Repository
	repo, err := requests.NewSQLiteRepository(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repo.Close()

	// 2. Initialize WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	// 3. Initialize Tunnel
	var tun tunnel.Tunnel
	if cfg.NoTunnel {
		tun = tunnel.NewMockTunnel("")
	} else {
		cfTun := tunnel.NewCloudflareTunnel()
		cfTun.OnStatus(func(status tunnel.Status, u string) {
			_ = hub.BroadcastJSON(map[string]any{
				"type":   "tunnel.status",
				"status": status,
				"url":    u,
			})
		})
		tun = cfTun
	}

	// 4. Initialize HTTP Inspection Proxy
	p, err := proxy.NewProxy(
		cfg.UpstreamURL(),
		repo,
		hub,
		cfg.MaxBodySize,
		cfg.RedactedHeaders,
	)
	if err != nil {
		log.Fatalf("Failed to initialize inspection proxy: %v", err)
	}

	// Start proxy listener
	proxyAddr := fmt.Sprintf("%s:%d", cfg.ProxyHost, cfg.ProxyPort)
	proxyListener, err := net.Listen("tcp", proxyAddr)
	if err != nil {
		log.Fatalf("Failed to bind proxy listener on %s: %v", proxyAddr, err)
	}
	proxyServer := &http.Server{Handler: p}
	go func() {
		if err := proxyServer.Serve(proxyListener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server error: %v", err)
		}
	}()
	defer func() {
		_ = proxyServer.Close()
	}()

	// 5. Initialize Dashboard Server with embedded assets
	staticFS, _ := web.Dist()
	dashServer := server.NewServer(cfg, repo, hub, tun, staticFS)
	go func() {
		if err := dashServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Dashboard server error: %v", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = dashServer.Stop(ctx)
	}()

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 6. Start Cloudflare Tunnel
	if !cfg.NoTunnel {
		fmt.Printf("Starting Cloudflare tunnel for %s...\n", cfg.ProxyURL())
		if err := tun.Start(ctx, cfg.ProxyURL()); err != nil {
			if err == tunnel.ErrCloudflaredNotFound {
				fmt.Fprintf(os.Stderr, "\n%s\n\n", tunnel.ErrCloudflaredNotFound)
				os.Exit(1)
			}
			if ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "Cloudflare tunnel error: %v\n", err)
			}
			return
		}
	}

	// Print beautiful status banner matching user spec
	printBanner(cfg, tun)

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("\nShutting down Tunnel Inspector...")
	_ = tun.Stop()
	fmt.Println("Done.")
}

func printBanner(cfg *config.Config, tun tunnel.Tunnel) {
	fmt.Println()
	fmt.Println("Tunnel Inspector")
	fmt.Println()
	fmt.Printf("✓ Application    %s\n", cfg.UpstreamURL())
	fmt.Printf("✓ Inspector      %s\n", cfg.DashboardURL())

	if cfg.NoTunnel {
		fmt.Printf("✓ Proxy          %s\n", cfg.ProxyURL())
		fmt.Println("\nPublic URL")
		fmt.Println("(Running in local --no-tunnel mode)")
	} else if tun.Status() == tunnel.StatusConnected {
		fmt.Println("✓ Cloudflared    connected")
		fmt.Println()
		fmt.Println("Public URL")
		fmt.Println(tun.URL())
	} else {
		fmt.Printf("! Cloudflared    %s\n", tun.Status())
	}

	fmt.Println()
	fmt.Println("Dashboard")
	fmt.Println(cfg.DashboardURL())
	fmt.Println()
	fmt.Println("Waiting for requests...")
}

func ensurePortAvailable(host string, defaultPort int) int {
	for port := defaultPort; port < defaultPort+100; port++ {
		addr := fmt.Sprintf("%s:%d", host, port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			_ = l.Close()
			return port
		}
	}
	return defaultPort
}
