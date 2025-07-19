package beacon

import (
	"sneakyfd/config"

	"context"

	"golang.org/x/sys/unix"
)

type contextKey string

const (
	BEACON_CONTEXT_KEY contextKey = "beacon"
)

type BeaconSender struct {
	fd int
}

func (b *BeaconSender) Send(beaconType BeaconType) (err error) {
	beacon := append(config.BeaconMagic, byte(beaconType))
	_, err = unix.Write(b.fd, beacon)
	return
}

func NewBeaconSender(fd int) *BeaconSender {
	return &BeaconSender{fd: fd}
}

func (b *BeaconSender) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, BEACON_CONTEXT_KEY, b)
}

func Ctx(ctx context.Context) *BeaconSender {
	v := ctx.Value(BEACON_CONTEXT_KEY)
	if b, ok := v.(*BeaconSender); ok {
		return b
	}
	return nil
}
