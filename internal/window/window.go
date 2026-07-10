// Package window parses the --window flag into a resolved time/rev range spec.
//
// A window is either a "daterange" (git --since/--until dates over a tip) or a
// "range" (an explicit git revision range like a..b, or the full history).
// Parsing is strict: an unrecognized form is a hard error, never a silent
// fallthrough.
package window

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Spec is a resolved time window.
type Spec struct {
	Mode     string // "range" | "daterange"
	RevRange string // e.g. "a..b"; "" when unused
	Since    string // git-approxidate / RFC3339; "" when unused
	Until    string
	Label    string
	Raw      string
}

// dateAddRE matches the "<YYYY-MM-DD>+<N><unit>" form.
var dateAddRE = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\+(\d+)(d|day|days|w|week|weeks|m|month|months|y|year|years)$`)

// rangeRE detects the "<refA>..<refB>" form.
var rangeRE = regexp.MustCompile(`\.\.`)

// ParseWindow resolves a window spec string. Forms are matched in order; an
// unrecognized form returns an error listing the supported forms.
func ParseWindow(spec string) (Spec, error) {
	now := time.Now()

	switch spec {
	case "today":
		start := startOfDay(now)
		return Spec{
			Mode:  "daterange",
			Since: start.Format(time.RFC3339),
			Label: fmt.Sprintf("Today (%s)", start.Format("2006-01-02")),
			Raw:   spec,
		}, nil
	case "yesterday":
		todayStart := startOfDay(now)
		yStart := todayStart.AddDate(0, 0, -1)
		return Spec{
			Mode:  "daterange",
			Since: yStart.Format(time.RFC3339),
			Until: todayStart.Format(time.RFC3339),
			Label: fmt.Sprintf("Yesterday (%s)", yStart.Format("2006-01-02")),
			Raw:   spec,
		}, nil
	case "week", "7d":
		return Spec{
			Mode:  "daterange",
			Since: now.AddDate(0, 0, -7).Format(time.RFC3339),
			Label: "Last 7 days",
			Raw:   spec,
		}, nil
	case "month", "30d":
		return Spec{
			Mode:  "daterange",
			Since: now.AddDate(0, 0, -30).Format(time.RFC3339),
			Label: "Last 30 days",
			Raw:   spec,
		}, nil
	case "all", "start", "start..now":
		return Spec{
			Mode:  "range",
			Label: "Full history",
			Raw:   spec,
		}, nil
	}

	if m := dateAddRE.FindStringSubmatch(spec); m != nil {
		startDate, err := time.Parse("2006-01-02", m[1])
		if err != nil {
			return Spec{}, fmt.Errorf("invalid date in window %q: %w", spec, err)
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return Spec{}, fmt.Errorf("invalid count in window %q: %w", spec, err)
		}
		var until time.Time
		switch m[3] {
		case "d", "day", "days":
			until = startDate.AddDate(0, 0, n)
		case "w", "week", "weeks":
			until = startDate.AddDate(0, 0, 7*n)
		case "m", "month", "months":
			until = startDate.AddDate(0, n, 0)
		case "y", "year", "years":
			until = startDate.AddDate(n, 0, 0)
		}
		return Spec{
			Mode:  "daterange",
			Since: startDate.Format("2006-01-02"),
			Until: until.Format("2006-01-02"),
			Label: fmt.Sprintf("%s → %s", startDate.Format("2006-01-02"), until.Format("2006-01-02")),
			Raw:   spec,
		}, nil
	}

	if rangeRE.MatchString(spec) {
		return Spec{
			Mode:     "range",
			RevRange: spec,
			Label:    spec,
			Raw:      spec,
		}, nil
	}

	return Spec{}, fmt.Errorf("unrecognized window %q; supported forms: today | yesterday | week | month | all | <YYYY-MM-DD>+<N><unit> (unit d/w/m/y) | <refA>..<refB>", spec)
}

// startOfDay returns midnight local time for the given moment.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
