package alerts

var keywordAssetKinds = map[string][]string{
	"satellite": {"satellite"},
	"space":     {"satellite"},
	"orbit":     {"satellite"},
	"RF":        {"ground_station"},
	"TM/TC":     {"ground_station"},
	"firmware":  {"firmware"},
}

func AssetKindsForKeyword(keyword string) []string {
	return keywordAssetKinds[keyword]
}
