package server

import "fmt"

type ResponderFlag struct {
	value  string
	expect []string
}

func (r *ResponderFlag) Set(s string) error {
	for _, e := range r.expect {
		if e == s {
			r.value = s
			return nil
		}
	}

	return fmt.Errorf("expecting one of %s", r.expect)
}
func (r ResponderFlag) String() string {
	return r.value
}
func (r ResponderFlag) Get() interface{} {
	return r.value
}
