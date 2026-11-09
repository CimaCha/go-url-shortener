package audit

import (
	"errors"
	"fmt"
	"net/http"

	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugins/redirect"
)

var (
	// ErrSendRequest indicates that an HTTP audit delivery failed.
	ErrSendRequest = errors.New("send audit request error")
)

// URLClientSubscriber posts audit events as JSON to a remote endpoint.
type URLClientSubscriber struct {
	id          string
	url         string
	auditClient *gentleman.Client
}

// NewURLClientSubscriber creates an HTTP audit observer with the given ID.
func NewURLClientSubscriber(id string, url string, auditClient *gentleman.Client) *URLClientSubscriber {
	return &URLClientSubscriber{
		id:          id,
		url:         url,
		auditClient: auditClient,
	}
}

func (s *URLClientSubscriber) update(exposure Exposure) error {
	return s.sendAudit(exposure)
}

func (s *URLClientSubscriber) getID() string {
	return s.id
}

func (s *URLClientSubscriber) sendAudit(exposure Exposure) error {
	response, err := s.auditClient.Post().URL(s.url).JSON(exposure).Use(redirect.Limit(1)).Send()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSendRequest, err)
	}
	defer response.RawResponse.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: HTTP %d", ErrSendRequest, response.StatusCode)
	}
	return nil
}
