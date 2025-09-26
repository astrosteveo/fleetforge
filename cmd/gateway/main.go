package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/astrosteveo/fleetforge/pkg/gateway"
)

// SimpleLogger implements the gateway.Logger interface
type SimpleLogger struct {
	debug bool
}

func NewSimpleLogger(debug bool) *SimpleLogger {
	return &SimpleLogger{debug: debug}
}

func (l *SimpleLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("[INFO] %s %v", msg, keysAndValues)
}

func (l *SimpleLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.Printf("[ERROR] %s: %v %v", msg, err, keysAndValues)
}

func (l *SimpleLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l.debug {
		log.Printf("[DEBUG] %s %v", msg, keysAndValues)
	}
}

func main() {
	var (
		port           = flag.Int("port", 8090, "Port to listen on")
		host           = flag.String("host", "0.0.0.0", "Host to bind to")
		debug          = flag.Bool("debug", false, "Enable debug logging")
		rateLimit      = flag.Int("rate-limit", 100, "Requests per second rate limit")
		burstSize      = flag.Int("burst-size", 20, "Rate limit burst size")
		sessionTimeout = flag.Duration("session-timeout", 5*time.Minute, "Session timeout duration")
		readTimeout    = flag.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
		writeTimeout   = flag.Duration("write-timeout", 30*time.Second, "HTTP write timeout")
	)
	flag.Parse()

	// Override with environment variables if present
	if envPort := os.Getenv("GATEWAY_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}

	if envHost := os.Getenv("GATEWAY_HOST"); envHost != "" {
		*host = envHost
	}

	if os.Getenv("DEBUG") == "true" {
		*debug = true
	}

	// Create logger
	logger := NewSimpleLogger(*debug)

	// Create gateway configuration
	config := gateway.DefaultGatewayConfig()
	config.Port = *port
	config.Host = *host
	config.ReadTimeout = *readTimeout
	config.WriteTimeout = *writeTimeout
	config.SessionTimeout = *sessionTimeout
	config.RateLimit.RequestsPerSecond = *rateLimit
	config.RateLimit.BurstSize = *burstSize

	logger.Info("starting FleetForge Gateway",
		"version", "1.0.0",
		"port", config.Port,
		"host", config.Host,
		"debug", *debug,
		"rateLimit", config.RateLimit.RequestsPerSecond,
		"sessionTimeout", config.SessionTimeout)

	// Create gateway server
	gatewayServer := gateway.NewGatewayServer(config, logger)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- gatewayServer.Start()
	}()

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil {
			logger.Error(err, "server startup failed")
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig)

		// Attempt graceful shutdown
		if err := gatewayServer.Stop(); err != nil {
			logger.Error(err, "graceful shutdown failed")
			os.Exit(1)
		}

		logger.Info("gateway shutdown completed")
	}
}
