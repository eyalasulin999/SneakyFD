package codec

import (
	"encoding/binary"
	"fmt"
	"io"

	"sneakyfd/protobuf/messages"

	"google.golang.org/protobuf/proto"
)

func ReadEnvelope(transport io.Reader) (envelope *messages.Envelope, err error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32

	n, err := transport.Read(dataLengthBuf)
	if err != nil {
		return
	}
	if n != 4 {
		err = fmt.Errorf("message lenght read error: expected 4 bytes, got %d", n)
		return
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	if dataLength <= 0 {
		err = fmt.Errorf("zero data length")
		return
	}

	dataBuf := make([]byte, dataLength)

	n, err = transport.Read(dataBuf)

	if err != nil {
		return
	}
	if n != dataLength {
		err = fmt.Errorf("message read error: expected %d bytes, got %d", dataLength, n)
		return
	}

	// Unmarshal the protobuf envelope
	envelope = &messages.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		return
	}

	return
}

func writeFull(w io.Writer, buf []byte) error {
	for len(buf) > 0 {
		n, err := w.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}

func WriteEnvelope(transport io.Writer, env *messages.Envelope) (err error) {
	// Marshal the envelope to protobuf bytes
	data, err := proto.Marshal(env)
	if err != nil {
		return
	}

	// Prepare length prefix (4 bytes, little-endian)
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(data)))

	// Write header
	err = writeFull(transport, header)
	if err != nil {
		return
	}

	// Write protobuf data
	err = writeFull(transport, data)
	if err != nil {
		return
	}

	return
}

func WrapEnvelope(msgType uint32, msg proto.Message) (envelope *messages.Envelope, err error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		return
	}

	envelope = &messages.Envelope{Type: msgType, Data: data}
	return
}
