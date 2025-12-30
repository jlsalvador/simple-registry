package mapset

type MapSet map[string]struct{}

func NewMapSet() MapSet {
	return map[string]struct{}{}
}

func (s MapSet) Add(d string) {
	s[d] = struct{}{}
}

func (s MapSet) Contains(d string) bool {
	_, ok := s[d]
	return ok
}
