package audit

// Action identifies the audited HTTP operation.
type Action string

const (
	// Follow records a successful redirect.
	Follow Action = "follow"
	// Shorten records a successful shortening request.
	Shorten Action = "shorten"
)

// Exposure is an audit event for a successful URL operation.
type Exposure struct {
	Timestamp int64
	Action    Action
	UserID    string
	URL       string
}
