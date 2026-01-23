package pipeline

import (
	"io"
)

type Transform func([]byte) ([]byte, error)

type Pipeline struct {
	transport io.ReadWriter
	inbound   []Transform
	outbound  []Transform
}

func NewPipeline(transport io.ReadWriter, inbound []Transform, outbound []Transform) Pipeline {
	return Pipeline{transport: transport, inbound: inbound, outbound: outbound}
}

func (p *Pipeline) Write(data []byte) (n int, err error) {
	for _, t := range p.outbound {
		data, err = t(data)
		if err != nil {
			return
		}
	}
	n, err = p.transport.Write(data)
	return
}

func (p *Pipeline) Read(buf []byte) (n int, err error) {
	n, err = p.transport.Read(buf)
	if err != nil {
		return
	}

	out := buf[:n]
	for _, t := range p.inbound {
		out, err = t(out)
		if err != nil {
			return
		}
	}

	n = copy(buf, out)
	return
}
