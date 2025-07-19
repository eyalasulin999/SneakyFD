package marker

type Marker interface {
	Check(fd int) (marked bool)
}

type Markers []Marker

func (m Markers) Check(fd int) (marked bool) {
	marked = true
	for _, m := range m {
		if !m.Check(fd) {
			marked = false
		}
	}
	return
}
