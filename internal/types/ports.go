package types

type Port interface {
	Equals(port uint16) bool
}

type Ports []Port

func (p Ports) Contains(port uint16) (contains bool) {
	for _, p := range p {
		if p.Equals(port) {
			contains = true
		}
	}
	return
}

type FixedPort struct {
	Port uint16
}

func (p FixedPort) Equals(port uint16) (equals bool) {
	equals = p.Port == port
	return
}

type RangePort struct {
	MinPort uint16
	MaxPort uint16
}

func (p RangePort) Equals(port uint16) (equals bool) {
	equals = port >= p.MinPort && port <= p.MaxPort
	return
}
