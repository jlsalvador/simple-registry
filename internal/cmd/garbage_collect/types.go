package garbagecollect

type digestSet map[string]struct{}

func newDigestSet() digestSet {
	return make(map[string]struct{})
}

func (s digestSet) add(d string) {
	s[d] = struct{}{}
}

func (s digestSet) has(d string) bool {
	_, ok := s[d]
	return ok
}
