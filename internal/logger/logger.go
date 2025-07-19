package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Log zerolog.Logger

func Init() {
	Log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	log.Logger = Log
}

func WithSessionID(sessionID string) zerolog.Logger {
	return Log.With().Str("session_id", sessionID).Logger()
}
