package echoserver

import (
	"context"

	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

func EchoServer(ctx context.Context, fd int) {
	log := zerolog.Ctx(ctx)

	buf := make([]byte, 1024)
	for {
		n, err := unix.Read(fd, buf)
		if err != nil {
			log.Error().Err(err).Msg("Read error")
			return
		}
		if n == 0 {
			log.Info().Msg("Connection closed by peer")
			return
		}

		data := buf[:n]
		log.Info().Str("data", string(data)).Msg("Received")

		// echo back
		_, err = unix.Write(fd, data)
		if err != nil {
			log.Error().Err(err).Msg("Write error")
			return
		}
	}
}
