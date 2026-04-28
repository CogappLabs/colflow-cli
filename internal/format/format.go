package format

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	Bold    = color.New(color.Bold).SprintFunc()
	Gray    = color.New(color.FgHiBlack).SprintFunc()
	Red     = color.New(color.FgRed).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	BoldRed = color.New(color.FgRed, color.Bold).SprintFunc()
)

func ColorStatus(status string) string {
	switch status {
	case "SUCCESS":
		return Green(status)
	case "FAILURE":
		return Red(status)
	case "STARTED", "STARTING":
		return Yellow(status)
	case "QUEUED":
		return Blue(status)
	default:
		return Gray(status)
	}
}

func TimeAgo(ts any) string {
	var seconds int64

	switch v := ts.(type) {
	case nil:
		return "never"
	case float64:
		if v == 0 {
			return "never"
		}
		seconds = int64(time.Now().Unix() - int64(v))
	case int64:
		if v == 0 {
			return "never"
		}
		seconds = time.Now().Unix() - v
	case string:
		if v == "" {
			return "never"
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "never"
		}
		// Dagster timestamp strings are milliseconds (some) or seconds (others).
		// Heuristic: if > 10^12, treat as ms.
		if f > 1e12 {
			f = f / 1000
		}
		seconds = time.Now().Unix() - int64(f)
	default:
		return "never"
	}

	if seconds < 0 {
		return "just now"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds ago", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm ago", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh ago", seconds/3600)
	}
	return fmt.Sprintf("%dd ago", seconds/86400)
}

func PadRight(s string, n int) string {
	// strip ANSI for length count
	visible := stripANSI(s)
	if len(visible) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(visible))
}

func stripANSI(s string) string {
	out := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}
