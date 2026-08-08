package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/idna"

	"monitoring-platform/packages/shared/domain"
)

type DomainExpirationExecutor struct {
	deps       Deps
	httpClient *http.Client
	cache      *domainLookupCache
}

func NewDomainExpirationExecutor(deps Deps) *DomainExpirationExecutor {
	transport := deps.Guard.WrapTransport(&http.Transport{
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true,
	})

	return &DomainExpirationExecutor{
		deps:       deps,
		httpClient: &http.Client{Transport: transport},
		cache:      newDomainLookupCache(time.Hour),
	}
}

func (e *DomainExpirationExecutor) Type() domain.MonitorType {
	return domain.MonitorDomainExpiration
}

func (e *DomainExpirationExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	result := newBaseResult(job)

	asciiDomain, err := idna.Lookup.ToASCII(strings.TrimSuffix(strings.ToLower(job.Target), "."))
	if err != nil {
		return finishFailure(result, "invalid_domain", fmt.Errorf("invalid domain %q: %w", job.Target, err))
	}

	lookupStart := time.Now()
	info, lookupErr := e.lookup(ctx, asciiDomain)
	lookupDuration := time.Since(lookupStart)

	result.Metrics["lookup_duration_ms"] = lookupDuration.Milliseconds()
	result.Attributes["domain"] = asciiDomain

	var notRegistered *errDomainNotRegistered
	if errors.As(lookupErr, &notRegistered) {
		result.Metrics["registered"] = 0
		return finishFailure(result, "domain_not_registered", lookupErr)
	}
	if lookupErr != nil {
		return finishFailure(result, "rdap_lookup_failed", lookupErr)
	}

	result.Metrics["registered"] = 1
	result.Attributes["source"] = info.Source
	result.Attributes["registrar"] = info.Registrar
	result.Attributes["statuses"] = info.Statuses
	result.Attributes["nameservers"] = info.Nameservers
	if info.CreatedAt != nil {
		result.Attributes["created_at"] = info.CreatedAt.UTC()
	}
	if info.UpdatedAt != nil {
		result.Attributes["updated_at"] = info.UpdatedAt.UTC()
	}

	if info.ExpiresAt == nil {
		return finishFailure(result, "expiration_date_missing", fmt.Errorf("registry did not provide an expiration date"))
	}

	expiresAt := info.ExpiresAt.UTC()
	daysRemaining := int(time.Until(expiresAt).Hours() / 24)

	result.Attributes["expires_at"] = expiresAt
	result.Attributes["days_remaining"] = daysRemaining
	result.Metrics["days_remaining"] = daysRemaining

	warningDays := intConfig(job.Config, "warning_days", 45)
	criticalDays := intConfig(job.Config, "critical_days", 15)

	if daysRemaining < 0 {
		return finishFailure(result, "domain_expired", fmt.Errorf("domain expired at %s", expiresAt.Format(time.RFC3339)))
	}

	if expectedRegistrar := stringConfig(job.Config, "expected_registrar_contains", ""); expectedRegistrar != "" {
		if !strings.Contains(strings.ToLower(info.Registrar), strings.ToLower(expectedRegistrar)) {
			return finishFailure(
				result,
				"registrar_mismatch",
				fmt.Errorf("registrar %q does not contain %q", info.Registrar, expectedRegistrar),
			)
		}
	}

	if boolConfig(job.Config, "check_nameservers", false) {
		expectedNameservers := stringSliceConfig(job.Config, "expected_nameservers", nil)
		if len(expectedNameservers) > 0 {
			matched := false
			for _, expected := range expectedNameservers {
				if containsString(info.Nameservers, strings.ToLower(strings.TrimSuffix(expected, "."))) {
					matched = true
					break
				}
			}
			result.Metrics["nameserver_match"] = boolToInt(matched)
			if !matched {
				return finishFailure(
					result,
					"nameserver_mismatch",
					fmt.Errorf("nameservers %v do not include any of %v", info.Nameservers, expectedNameservers),
				)
			}
		}
	}

	if daysRemaining <= criticalDays {
		return finishFailure(
			result,
			"domain_expiring",
			fmt.Errorf("domain expires in %d days (critical threshold %d)", daysRemaining, criticalDays),
		)
	}

	if daysRemaining <= warningDays {
		result.Attributes["expiry_warning"] = true
	}

	return finishSuccess(result)
}

func (e *DomainExpirationExecutor) lookup(ctx context.Context, domainName string) (domainInfo, error) {
	if info, cachedErr, ok := e.cache.get(domainName); ok {
		return info, cachedErr
	}

	info, err := lookupRDAP(ctx, e.httpClient, domainName)
	if err != nil {
		var notRegistered *errDomainNotRegistered
		if !errors.As(err, &notRegistered) {
			if whoisInfo, whoisErr := lookupWHOIS(ctx, e.deps.Guard.DialContext, domainName); whoisErr == nil {
				e.cache.set(domainName, whoisInfo, nil)
				return whoisInfo, nil
			} else if errors.As(whoisErr, &notRegistered) {
				e.cache.set(domainName, domainInfo{}, whoisErr)
				return domainInfo{}, whoisErr
			}
		}

		e.cache.set(domainName, domainInfo{}, err)
		return domainInfo{}, err
	}

	e.cache.set(domainName, info, nil)
	return info, nil
}
