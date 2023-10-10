package main

import (
	"context"
	"errors"
	"github.com/spf13/cobra"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const DEFAULT_HOST = "0.0.0.0"
const DEFAULT_TRUNK_SIZE = 1024

var _version = "not set"
var _usingUDP = false
var _port = 0
var _host = DEFAULT_HOST
var _trunkSize = DEFAULT_TRUNK_SIZE

type Listener interface {
	Listen() error
}

func startServer(ctx context.Context, canceler context.CancelFunc) {
	err := NewTCPListener(_host, _port, ctx).Listen()
	if err != nil {
		canceler()
	}
}

func run() {
	ctx, canceler := context.WithCancel(context.Background())
	go startServer(ctx, canceler)
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-c:
			canceler()
			continue
		case <-ctx.Done():
			return
		}
	}
}

var _commandRoot = cobra.Command{
	Use:     "listenndump",
	Version: _version,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _trunkSize < 16 {
			return errors.New("trunk size must be greater than 16")
		}
		run()
		return nil
	},
}

func initArgs() {
	_commandRoot.Flags().BoolVarP(&_usingUDP, "udp", "u", false, "Use UDP instead of TCP")
	_commandRoot.Flags().IntVarP(&_port, "port", "p", 0, "Port to listen")
	_commandRoot.Flags().StringVarP(&_host, "host", "o", DEFAULT_HOST, "Host to listen")
	_commandRoot.Flags().IntVarP(&_trunkSize, "trunk", "t", DEFAULT_TRUNK_SIZE, "Trunk size")
	err := _commandRoot.MarkFlagRequired("port")
	if err != nil {
		slog.Error("Failed to mark flag as required", err)
		os.Exit(1)
		return
	}
}

func main() {
	initArgs()
	err := _commandRoot.Execute()
	if err != nil {
		slog.Debug("Failed to execute", err)
	}
}
