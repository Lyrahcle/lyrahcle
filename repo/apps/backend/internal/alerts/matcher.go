// Package alerts matches newly synced space-related CVEs to the assets
// they're relevant to, and records that match as an alert an operator can
// review and acknowledge.
package alerts

// keywordAssetKinds maps a cve-sync search keyword (see
// apps/cve-sync/internal/config/config.go's SyncKeywords) to the asset
// kinds (see NewAssetForm.jsx's KINDS) it's relevant to. "domain" and "ip"
// assets have no entry here — there's no NVD search keyword that
// meaningfully signals relevance to a generic domain or IP asset, so they
// never get matched rather than being matched against everything.
var keywordAssetKinds = map[string][]string{
	"satellite": {"satellite"},
	"space":     {"satellite"},
	"orbit":     {"satellite"},
	"RF":        {"ground_station"},
	"TM/TC":     {"ground_station"},
	"firmware":  {"firmware"},
}

// AssetKindsForKeyword returns which asset kinds a CVE synced under
// keyword is relevant to. An empty result means "no known relevance" —
// not an error.
func AssetKindsForKeyword(keyword string) []string {
	return keywordAssetKinds[keyword]
}
