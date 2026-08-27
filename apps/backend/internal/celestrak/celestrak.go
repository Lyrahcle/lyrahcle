package celestrak

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	NoradID int
	Name    string
	Line1   string
	Line2   string
}

func FetchGroup(group string) ([]Entry, error) {
	url := fmt.Sprintf("https://celestrak.org/NORAD/elements/gp.php?GROUP=%s&FORMAT=tle", group)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("celestrak: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("celestrak: unexpected status %d", resp.StatusCode)
	}

	return parseTLE(resp.Body)
}

func parseTLE(r interface{ Read([]byte) (int, error) }) ([]Entry, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("celestrak: scan: %w", err)
	}

	var entries []Entry
	for i := 0; i+2 < len(lines); i += 3 {
		name := lines[i]
		line1 := lines[i+1]
		line2 := lines[i+2]

		if !strings.HasPrefix(line1, "1 ") || !strings.HasPrefix(line2, "2 ") {
			continue
		}

		noradID, err := strconv.Atoi(strings.TrimSpace(line1[2:7]))
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			NoradID: noradID,
			Name:    strings.TrimSpace(name),
			Line1:   line1,
			Line2:   line2,
		})
	}

	return entries, nil
}
