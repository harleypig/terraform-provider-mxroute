package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// This file is the canonical Go representation of the MXroute API's data
// models — one type per OpenAPI component schema, plus the inline response
// shapes the endpoints without a named schema return. Resources and data
// sources unmarshal API responses into these types (via client.Do) instead
// of declaring their own ad-hoc structs, so the model layer stays a single
// source of truth that mirrors the spec. A nullable field is a pointer.

// Domain is a mail domain on the account (GET /domains/{domain}). Pointers is
// always the list of pointer names, decoded tolerantly — see UnmarshalJSON.
type Domain struct {
	Domain      string   `json:"domain"`
	MailHosting bool     `json:"mail_hosting"`
	SSLEnabled  bool     `json:"ssl_enabled"`
	Pointers    []string `json:"pointers"`
}

// UnmarshalJSON decodes a Domain, tolerating either shape the API uses for
// pointers. The OpenAPI spec declares an array of strings, but the live
// GET /domains/{domain} returns an object keyed by pointer name once the
// domain has any pointer — decoding that into []string previously failed with
// "cannot unmarshal object into Go struct field Domain.pointers". Either
// shape reduces to the list of pointer names. See API-MAPPING.md.
func (d *Domain) UnmarshalJSON(data []byte) error {
	// A distinct type breaks the recursion into this method; Pointers is
	// pulled out as raw JSON and decoded by shape below.
	type domainAlias struct {
		Domain      string          `json:"domain"`
		MailHosting bool            `json:"mail_hosting"`
		SSLEnabled  bool            `json:"ssl_enabled"`
		Pointers    json.RawMessage `json:"pointers"`
	}

	var raw domainAlias

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	names, err := decodePointerNames(raw.Pointers)
	if err != nil {
		return err
	}

	d.Domain = raw.Domain
	d.MailHosting = raw.MailHosting
	d.SSLEnabled = raw.SSLEnabled
	d.Pointers = names

	return nil
}

// decodePointerNames extracts the pointer names from the two shapes the API
// uses for a domain's pointers: a JSON array of strings (the spec) or a JSON
// object keyed by pointer name (the live response). An absent or null value
// yields no names. Object keys are sorted so state stays deterministic.
func decodePointerNames(raw json.RawMessage) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)

	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}

	switch trimmed[0] {
	case '[':
		var names []string

		if err := json.Unmarshal(trimmed, &names); err != nil {
			return nil, err
		}

		return names, nil

	case '{':
		var obj map[string]json.RawMessage

		if err := json.Unmarshal(trimmed, &obj); err != nil {
			return nil, err
		}

		names := make([]string, 0, len(obj))

		for name := range obj {
			names = append(names, name)
		}

		sort.Strings(names)

		return names, nil

	default:
		return nil, fmt.Errorf("unexpected pointers shape in domain response: %s", trimmed)
	}
}

// EmailAccount is a mailbox (GET /domains/{domain}/email-accounts/{user}).
type EmailAccount struct {
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Quota     int64   `json:"quota"` // megabytes; 0 = unlimited
	Usage     float64 `json:"usage"` // megabytes currently used
	Limit     int64   `json:"limit"` // daily outbound send limit
	Sent      int64   `json:"sent"`  // messages sent today
	Suspended bool    `json:"suspended"`
}

// Forwarder is an email forwarder (GET /domains/{domain}/forwarders).
type Forwarder struct {
	Alias        string   `json:"alias"`
	Email        string   `json:"email"`
	Destinations []string `json:"destinations"`
}

// DomainPointer is a domain alias or redirect
// (GET /domains/{domain}/pointers).
type DomainPointer struct {
	Pointer string `json:"pointer"`
	Type    string `json:"type"` // "alias" or "redirect"
	Target  string `json:"target"`
}

// DNSInfo is the DNS configuration for a domain
// (GET /domains/{domain}/dns). DKIM and Verification are absent on some
// domains, so they are nullable.
type DNSInfo struct {
	MXRecords    []DNSMXRecord    `json:"mx_records"`
	SPF          DNSRecord        `json:"spf"`
	DKIM         *DNSRecord       `json:"dkim"`
	Verification *DNSVerification `json:"verification"`
}

// DNSMXRecord is one MX record within DNSInfo.
type DNSMXRecord struct {
	Priority    int64  `json:"priority"`
	Hostname    string `json:"hostname"`
	Description string `json:"description"`
}

// DNSRecord is a single DNS record (SPF or DKIM) within DNSInfo.
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DNSVerification is the domain-verification record within DNSInfo.
type DNSVerification struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// CatchAll is a domain's catch-all policy
// (GET /domains/{domain}/catch-all). Address is set only when Type is
// "address".
type CatchAll struct {
	Type        string  `json:"type"` // "fail", "blackhole", or "address"
	Address     *string `json:"address"`
	Description string  `json:"description"`
}

// SpamSettings is a domain's spam configuration
// (GET /domains/{domain}/spam/settings).
type SpamSettings struct {
	HighScore int64 `json:"high_score"` // auto-delete score threshold (1-50)
}

// Quota is the account-wide storage quota (GET /quota).
type Quota struct {
	Username    string            `json:"username"`
	TotalUsed   int64             `json:"total_used"`  // bytes
	TotalLimit  int64             `json:"total_limit"` // bytes; 0 = unlimited
	PercentUsed float64           `json:"percent_used"`
	Breakdown   QuotaBreakdown    `json:"breakdown"`
	GracePeriod *QuotaGracePeriod `json:"grace_period"` // present only when over quota
	UpdatedAt   string            `json:"updated_at"`
}

// QuotaBreakdown is per-category usage in bytes within Quota.
type QuotaBreakdown struct {
	Email     int64 `json:"email"`
	Web       int64 `json:"web"`
	Databases int64 `json:"databases"`
	Backups   int64 `json:"backups"`
	Other     int64 `json:"other"`
}

// QuotaGracePeriod is set on Quota when the account has exceeded its limit.
type QuotaGracePeriod struct {
	DaysRemaining int64  `json:"days_remaining"`
	Deadline      string `json:"deadline"`
}

// EmailQuota is per-mailbox usage (GET /quota/email).
type EmailQuota struct {
	Username string              `json:"username"`
	Accounts []EmailQuotaAccount `json:"accounts"`
}

// EmailQuotaAccount is one mailbox's usage within EmailQuota.
type EmailQuotaAccount struct {
	EmailAddress string `json:"email_address"`
	SizeBytes    int64  `json:"size_bytes"`
	UpdatedAt    string `json:"updated_at"`
}

// VerificationKey is the account ownership-verification record
// (GET /verification-key).
type VerificationKey struct {
	Key         string             `json:"key"`
	Record      VerificationRecord `json:"record"`
	Description string             `json:"description"`
}

// VerificationRecord is the DNS record to publish, within VerificationKey.
type VerificationRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResellerUser is a reseller-managed user (GET /reseller/users/{username}).
type ResellerUser struct {
	Username  string        `json:"username"`
	Email     string        `json:"email"`
	Domain    string        `json:"domain"`
	Package   string        `json:"package"`
	Suspended bool          `json:"suspended"`
	Quota     ResellerQuota `json:"quota"`
}

// ResellerQuota is a reseller user's quota within ResellerUser.
type ResellerQuota struct {
	Limit     *int64  `json:"limit"` // null = unlimited
	Used      float64 `json:"used"`
	Unlimited bool    `json:"unlimited"`
}

// Package is a reseller package (GET /reseller/packages/{name}).
type Package struct {
	Name     string          `json:"name"`
	Settings PackageSettings `json:"settings"`
}

// PackageSettings are the limits a reseller Package grants; a null field
// means unlimited.
type PackageSettings struct {
	QuotaGB         *float64 `json:"quota_gb"`
	QuotaUnlimited  bool     `json:"quota_unlimited"`
	Domains         *int64   `json:"domains"`
	EmailAccounts   *int64   `json:"email_accounts"`
	EmailForwarders *int64   `json:"email_forwarders"`
	DomainPointers  *int64   `json:"domain_pointers"`
}
