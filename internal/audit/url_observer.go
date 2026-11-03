package audit

import "fmt"

type subscriber struct {
	id string
}

func (s *subscriber) update(desc string) {
	fmt.Printf("The %s subscriber is notified of the %s event\n", s.id, desc)
}

func (s *subscriber) getID() string {
	return s.id
}
