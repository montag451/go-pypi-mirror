package flagutil

import "fmt"

type StringSlice []string

func (ss *StringSlice) String() string {
	return fmt.Sprint(*ss)
}

func (ss *StringSlice) Set(s string) error {
	*ss = append(*ss, s)
	return nil
}
