package audit

// Observer receives audit events and supplies a registration ID.
type Observer interface {
	update(Exposure) error

	getID() string
}
