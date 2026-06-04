package query

import (
	"context"
	"net/http"
	"time"

	domainrepo "git.at.oechsler.it/samuel/dash/v2/domain/repo"
)

type ApplicationHealthStatus string

const (
	ApplicationHealthOnline  ApplicationHealthStatus = "online"
	ApplicationHealthWarning ApplicationHealthStatus = "warning"
	ApplicationHealthOffline ApplicationHealthStatus = "offline"
)

type ApplicationHealth struct {
	Status     ApplicationHealthStatus
	StatusCode *int
}

// ApplicationHealthChecker handles checking whether an application URL responds.
type ApplicationHealthChecker interface {
	Handle(ctx context.Context, id uint) (ApplicationHealth, error)
}

type ApplicationStatusProbe interface {
	Probe(ctx context.Context, url string) (int, error)
}

type HTTPApplicationStatusProbe struct {
	Client *http.Client
}

func NewHTTPApplicationStatusProbe() *HTTPApplicationStatusProbe {
	return &HTTPApplicationStatusProbe{
		Client: &http.Client{Timeout: 3 * time.Second},
	}
}

func (p *HTTPApplicationStatusProbe) Probe(ctx context.Context, url string) (int, error) {
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}

	statusCode, err := probeWithMethod(ctx, client, http.MethodHead, url)
	if err == nil && statusCode != http.StatusMethodNotAllowed {
		return statusCode, nil
	}

	return probeWithMethod(ctx, client, http.MethodGet, url)
}

func probeWithMethod(ctx context.Context, client *http.Client, method string, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

type CheckApplicationHealth struct {
	ApplicationRepo domainrepo.ApplicationRepository
	Probe           ApplicationStatusProbe
}

func NewCheckApplicationHealth(applicationRepo domainrepo.ApplicationRepository) *CheckApplicationHealth {
	return &CheckApplicationHealth{
		ApplicationRepo: applicationRepo,
		Probe:           NewHTTPApplicationStatusProbe(),
	}
}

func (h *CheckApplicationHealth) Handle(ctx context.Context, id uint) (ApplicationHealth, error) {
	app, err := h.ApplicationRepo.Get(ctx, id)
	if err != nil {
		return ApplicationHealth{}, err
	}

	statusCode, err := h.Probe.Probe(ctx, app.Url)
	if err != nil {
		return ApplicationHealth{Status: ApplicationHealthOffline}, nil
	}

	status := ApplicationHealthWarning
	if statusCode >= 200 && statusCode < 400 {
		status = ApplicationHealthOnline
	}

	return ApplicationHealth{
		Status:     status,
		StatusCode: &statusCode,
	}, nil
}
