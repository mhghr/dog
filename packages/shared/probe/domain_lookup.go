package probe

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// rdapDomain is the subset of the RDAP domain response the platform needs.
type rdapDomain struct {
	Events []struct {
		EventAction string    `json:"eventAction"`
		EventDate   time.Time `json:"eventDate"`
	} `json:"events"`
	Status   []string     `json:"status"`
	Entities []rdapEntity `json:"entities"`
	Nameservers []struct {
		LdhName string `json:"ldhName"`
	} `json:"nameservers"`
}

// rdapEntity is a registrar/registrant entity from an RDAP response.
type rdapEntity struct {
	Roles      []string        `json:"roles"`
	VcardArray json.RawMessage `json:"vcardArray"`
}

type domainInfo struct {
	Registered  bool
	ExpiresAt   *time.Time
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	Registrar   string
	Statuses    []string
	Nameservers []string
	Source      string
}

type domainCacheEntry struct {
	fetchedAt time.Time
	info      domainInfo
	err       error
}

// domainLookupCache protects registry services from aggressive re-querying.
type domainLookupCache struct {
	mu      sync.Mutex
	entries map[string]domainCacheEntry
	ttl     time.Duration
}

func newDomainLookupCache(ttl time.Duration) *domainLookupCache {
	return &domainLookupCache{
		entries: make(map[string]domainCacheEntry),
		ttl:     ttl,
	}
}

func (c *domainLookupCache) get(domainName string) (domainInfo, error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[domainName]
	if !ok || time.Since(entry.fetchedAt) > c.ttl {
		return domainInfo{}, nil, false
	}

	return entry.info, entry.err, true
}

func (c *domainLookupCache) set(domainName string, info domainInfo, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[domainName] = domainCacheEntry{
		fetchedAt: time.Now(),
		info:      info,
		err:       err,
	}
}

const rdapBootstrapURL = "https://rdap.org/domain/"

type errDomainNotRegistered struct{ domain string }

func (e *errDomainNotRegistered) Error() string {
	return fmt.Sprintf("domain %q is not registered", e.domain)
}

func lookupRDAP(ctx context.Context, client *http.Client, domainName string) (domainInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rdapBootstrapURL+domainName, nil)
	if err != nil {
		return domainInfo{}, err
	}
	request.Header.Set("Accept", "application/rdap+json")
	request.Header.Set("User-Agent", "MonitoringPlatform/1.0")

	response, err := client.Do(request)
	if err != nil {
		return domainInfo{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return domainInfo{}, &errDomainNotRegistered{domain: domainName}
	}

	if response.StatusCode >= 300 {
		return domainInfo{}, fmt.Errorf("RDAP lookup returned status %s", response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return domainInfo{}, err
	}

	var payload rdapDomain
	if err := json.Unmarshal(body, &payload); err != nil {
		return domainInfo{}, fmt.Errorf("decode RDAP response: %w", err)
	}

	return rdapInfoFromPayload(payload), nil
}

// rdapInfoFromPayload converts the parsed RDAP response into the domain info
// the platform consumes (dates, registrar, nameservers).
func rdapInfoFromPayload(payload rdapDomain) domainInfo {
	info := domainInfo{
		Registered: true,
		Statuses:   payload.Status,
		Source:     "rdap",
	}

	for _, event := range payload.Events {
		eventDate := event.EventDate
		switch strings.ToLower(event.EventAction) {
		case "expiration":
			info.ExpiresAt = &eventDate
		case "registration":
			info.CreatedAt = &eventDate
		case "last changed":
			info.UpdatedAt = &eventDate
		}
	}

	for _, entity := range payload.Entities {
		if name := rdapRegistrarName(entity); name != "" {
			info.Registrar = name
		}
	}

	for _, nameserver := range payload.Nameservers {
		if nameserver.LdhName != "" {
			info.Nameservers = append(info.Nameservers, strings.ToLower(strings.TrimSuffix(nameserver.LdhName, ".")))
		}
	}

	return info
}

// rdapRegistrarName returns the registrar's full name from an RDAP entity if
// the entity carries the registrar role.
func rdapRegistrarName(entity rdapEntity) string {
	for _, role := range entity.Roles {
		if strings.EqualFold(role, "registrar") {
			if name := parseVcardFullName(entity.VcardArray); name != "" {
				return name
			}
		}
	}
	return ""
}

// parseVcardFullName extracts the "fn" property from a jCard array.
func parseVcardFullName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var vcard []json.RawMessage
	if err := json.Unmarshal(raw, &vcard); err != nil || len(vcard) < 2 {
		return ""
	}

	var properties [][]json.RawMessage
	if err := json.Unmarshal(vcard[1], &properties); err != nil {
		return ""
	}

	for _, property := range properties {
		if len(property) < 4 {
			continue
		}

		var name string
		if err := json.Unmarshal(property[0], &name); err != nil || name != "fn" {
			continue
		}

		var value string
		if err := json.Unmarshal(property[3], &value); err == nil && value != "" {
			return value
		}
	}

	return ""
}

var (
	whoisExpiryPattern     = regexp.MustCompile(`(?i)(?:registry expiry date|expiry date|expiration date|expiration time|expires(?: on)?|paid-till)\s*:?\s*(.+)`)
	whoisRegistrarPattern  = regexp.MustCompile(`(?i)^\s*registrar\s*:?\s*(.+)$`)
	whoisNameserverPattern = regexp.MustCompile(`(?i)^\s*(?:name server|nserver)\s*:?\s*(\S+)`)
	whoisReferPattern      = regexp.MustCompile(`(?i)^\s*(?:refer|whois)\s*:?\s*(\S+)`)
)

var whoisDateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02-Jan-2006",
	"2006.01.02 15:04:05",
	"2006.01.02",
}

func lookupWHOIS(ctx context.Context, dial func(context.Context, string, string) (net.Conn, error), domainName string) (domainInfo, error) {
	referral, err := whoisQuery(ctx, dial, "whois.iana.org:43", domainName)
	if err != nil {
		return domainInfo{}, err
	}

	server := ""
	for _, line := range strings.Split(referral, "\n") {
		if match := whoisReferPattern.FindStringSubmatch(line); match != nil {
			server = strings.TrimSpace(match[1])
			break
		}
	}

	responseText := referral
	if server != "" && !strings.EqualFold(server, "whois.iana.org") {
		if text, err := whoisQuery(ctx, dial, net.JoinHostPort(server, "43"), domainName); err == nil {
			responseText = text
		}
	}

	info := whoisInfoFromResponse(responseText)

	lower := strings.ToLower(responseText)
	if strings.Contains(lower, "no match for") ||
		strings.Contains(lower, "not found") && info.ExpiresAt == nil && info.Registrar == "" {
		return domainInfo{}, &errDomainNotRegistered{domain: domainName}
	}

	info.Registered = true
	return info, nil
}

// whoisInfoFromResponse parses WHOIS response lines for expiry, registrar and
// nameserver fields.
func whoisInfoFromResponse(responseText string) domainInfo {
	info := domainInfo{Source: "whois"}

	for _, line := range strings.Split(responseText, "\n") {
		if info.ExpiresAt == nil {
			if match := whoisExpiryPattern.FindStringSubmatch(line); match != nil {
				if parsed, ok := parseWhoisDate(strings.TrimSpace(match[1])); ok {
					info.ExpiresAt = &parsed
				}
			}
		}

		if info.Registrar == "" {
			if match := whoisRegistrarPattern.FindStringSubmatch(line); match != nil {
				info.Registrar = strings.TrimSpace(match[1])
			}
		}

		if match := whoisNameserverPattern.FindStringSubmatch(line); match != nil {
			nameserver := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(match[1]), "."))
			if nameserver != "" && !containsString(info.Nameservers, nameserver) {
				info.Nameservers = append(info.Nameservers, nameserver)
			}
		}
	}

	return info
}

func whoisQuery(ctx context.Context, dial func(context.Context, string, string) (net.Conn, error), address, query string) (string, error) {
	conn, err := dial(ctx, "tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return "", err
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(io.LimitReader(conn, 1024*1024))
	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	return builder.String(), scanner.Err()
}

func parseWhoisDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range whoisDateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
