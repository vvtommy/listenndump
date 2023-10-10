package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
)

type TCPListener struct {
	port    int
	host    string
	context context.Context

	listener net.Listener
}

func handleTCPConnection(connection net.Conn) {
	var err error
	var size int
	firstTime := false
	dumper := hex.Dumper(os.Stdout)
	defer func() {
		_ = dumper.Close()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("Connection closed", slog.String("remote", connection.RemoteAddr().String()))
			} else {
				slog.Error("Failed to read from connection, disconnect", err, slog.String("remote", connection.RemoteAddr().String()))
			}
		}
		err := connection.Close()
		if err != nil {
			slog.Error("Failed to close connection", err)
		}
	}()
	for {
		buf := make([]byte, 128)
		size, err = connection.Read(buf)
		if err != nil {
			return
		}
		if !firstTime {
			slog.Info("Received data", slog.String("remote", connection.RemoteAddr().String()))
			firstTime = true
			fmt.Print("\n")
		}
		_, _ = dumper.Write(buf[:size])
	}

}

func (t *TCPListener) waitConnection(closeSign chan<- struct{}) {
	for {
		connection, err := t.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				slog.Info("Listener closed")
				closeSign <- struct{}{}
			}
			slog.Error("Failed to accept connection", err)
			return
		}
		slog.Info("Accepted connection", slog.String("remote", connection.RemoteAddr().String()))
		go handleTCPConnection(connection)
	}
}

func (t *TCPListener) Listen() error {
	address := fmt.Sprintf("%s:%d", t.host, t.port)
	slog.Info(fmt.Sprintf("Listening on %s", address))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		slog.Error("Failed to listen", slog.String("address", address), err)
		return err
	}

	defer listener.Close()
	t.listener = listener
	closeSign := make(chan struct{}, 1)
	go t.waitConnection(closeSign)
	<-t.context.Done()
	t.listener.Close()
	slog.Info("Shutting down listener")
	<-closeSign
	slog.Info("Listener closed")

	return nil
}

func NewTCPListener(host string, port int, context context.Context) Listener {
	return &TCPListener{
		host:    host,
		port:    port,
		context: context,
	}
}
