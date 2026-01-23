package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"os"

	"sneakyfd/config"
	"sneakyfd/internal/codec"
	"sneakyfd/internal/pipeline"
	"sneakyfd/internal/rpc"

	"github.com/rs/zerolog"
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

func handleBackdoor(ctx context.Context, fd int) {
	log := zerolog.Ctx(ctx)

	conn, err := fdToConn(fd)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Convert fd to net.Conn failed")
		return
	}

	// For future use
	p := pipeline.NewPipeline(conn, config.PipelineInbound, config.PipelineOutbound)

	for {
		envelope, err := codec.ReadEnvelope(&p)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Read envelope failed")
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				log.Info().Msg("Client disconnected")
				break
			}
			continue
		}

		log.Info().
			Interface("envelope", envelope).
			Msg("Received envelope")

		handler, ok := rpc.Handlers[envelope.Type]
		if !ok {
			log.Warn().Uint32("type", envelope.Type).Msg("Unknown message type")
			continue
		}

		resType, res, err := handler(envelope.Data)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Handle RPC failed")
			continue
		}

		resEnvelope, err := codec.WrapEnvelope(resType, res)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Wrap envelope failed")
			continue
		}

		err = codec.WriteEnvelope(&p, resEnvelope)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Write envelope failed")
			continue
		}

		log.Info().
			Interface("envelope", resEnvelope).
			Msg("Sent envelope")
	}
}
