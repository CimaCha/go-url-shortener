package audit

type Action string

const (
	Follow  Action = "follow"
	Shorten Action = "shorten"
)

type Exposure struct {
	Timestamp int64
	Action    Action
	UserID    string
	URL       string
}
