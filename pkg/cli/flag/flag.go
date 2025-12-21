package flag

import "fmt"

type FlagValueStringSlice struct {
	Slice []string
}

func (ss *FlagValueStringSlice) String() string {
	return fmt.Sprint(ss.Slice)
}

func (ss *FlagValueStringSlice) Set(value string) error {
	ss.Slice = append(ss.Slice, value)
	return nil
}
