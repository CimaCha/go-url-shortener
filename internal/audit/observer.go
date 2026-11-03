package audit

type Observer interface {
	update(Exposure) error

	getID() string
}
