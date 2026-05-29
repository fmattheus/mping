package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	probing "github.com/prometheus-community/pro-bing"
)

// secDuration is a flag.Value that accepts plain numbers as seconds (e.g. 5, 1.5)
// and falls back to Go duration strings (e.g. 500ms, 2m) when a unit is present.
type secDuration time.Duration

func (s *secDuration) String() string {
	secs := time.Duration(*s).Seconds()
	if secs == float64(int64(secs)) {
		return fmt.Sprintf("%.0f", secs)
	}
	return strconv.FormatFloat(secs, 'f', -1, 64)
}

func (s *secDuration) Set(val string) error {
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		*s = secDuration(time.Duration(f * float64(time.Second)))
		return nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fmt.Errorf("use seconds (e.g. 5, 1.5) or a duration string (e.g. 500ms, 2m)")
	}
	*s = secDuration(d)
	return nil
}

var permWarnOnce sync.Once

func isPermissionError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "operation not permitted") ||
		strings.Contains(s, "permission denied")
}

func warnPermission() {
	permWarnOnce.Do(func() {
		fmt.Fprintln(os.Stderr, "\nNote: ICMP socket permission denied — unprivileged ping is not allowed.")
		fmt.Fprintln(os.Stderr, "See https://github.com/fmattheus/mping#linux for setup instructions.\n")
	})
}

type PingResult struct {
	Host string
	Err  error
}

type themeConfig struct {
	ok   func(string) string
	fail func(string) string
}

func colorTheme(okColor, failColor *color.Color, withSymbols bool) themeConfig {
	okPfx, failPfx := "", ""
	if withSymbols {
		okPfx, failPfx = "✓ ", "✗ "
	}
	return themeConfig{
		ok:   func(h string) string { return okColor.Sprint(okPfx + h) },
		fail: func(h string) string { return failColor.Sprint(failPfx + h) },
	}
}

var themes = map[string]themeConfig{
	"default": colorTheme(
		color.New(color.FgGreen, color.Bold),
		color.New(color.FgRed, color.Bold),
		false,
	),
	"symbols": colorTheme(
		color.New(color.FgGreen, color.Bold),
		color.New(color.FgRed, color.Bold),
		true,
	),
	"colorblind": colorTheme(
		color.New(color.FgHiBlue, color.Bold),
		color.New(color.FgHiYellow, color.Bold),
		true,
	),
	"mono": {
		ok:   func(h string) string { return "✓ " + h },
		fail: func(h string) string { return "✗ " + h },
	},
}

func pingHost(ctx context.Context, host string, timeout time.Duration) PingResult {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return PingResult{Host: host, Err: err}
	}
	pinger.Count = 1
	pinger.Timeout = timeout
	pinger.SetPrivileged(false)
	if err := pinger.RunWithContext(ctx); err != nil {
		if isPermissionError(err) {
			warnPermission()
		}
		return PingResult{Host: host, Err: err}
	}
	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return PingResult{Host: host, Err: fmt.Errorf("timeout")}
	}
	return PingResult{Host: host}
}

func runCycle(ctx context.Context, hosts []string, timeout time.Duration) []PingResult {
	results := make([]PingResult, len(hosts))
	var wg sync.WaitGroup
	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()
			results[idx] = pingHost(ctx, h, timeout)
		}(i, host)
	}
	wg.Wait()
	return results
}

func renderCycle(results []PingResult, t time.Time, theme themeConfig) {
	fmt.Fprint(color.Output, t.Format("15:04:05"))
	for _, r := range results {
		fmt.Fprint(color.Output, " ")
		if r.Err != nil {
			fmt.Fprint(color.Output, theme.fail(r.Host))
		} else {
			fmt.Fprint(color.Output, theme.ok(r.Host))
		}
	}
	fmt.Fprintln(color.Output)
}

func main() {
	var interval secDuration = secDuration(5 * time.Second)
	var timeout secDuration
	var themeName string

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mping [flags] host [host ...]")
		fmt.Fprintln(os.Stderr, "  -i, --interval seconds   time between ping cycles (default 5)")
		fmt.Fprintln(os.Stderr, "  -t, --timeout  seconds   per-ping timeout (default: 90% of interval)")
		fmt.Fprintln(os.Stderr, "  -T, --theme    name      output theme (default: default)")
		fmt.Fprintln(os.Stderr, "                           themes: default, symbols, colorblind, mono")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Durations accept plain seconds (5, 1.5) or duration strings (500ms, 2m).")
	}

	flag.Var(&interval, "interval", "time between ping cycles in seconds")
	flag.Var(&interval, "i", "")
	flag.Var(&timeout, "timeout", "per-ping timeout in seconds")
	flag.Var(&timeout, "t", "")
	flag.StringVar(&themeName, "theme", "default", "output theme")
	flag.StringVar(&themeName, "T", "default", "")
	flag.Parse()

	theme, ok := themes[themeName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown theme %q; available: default, symbols, colorblind, mono\n", themeName)
		os.Exit(1)
	}

	var timeoutSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "timeout" || f.Name == "t" {
			timeoutSet = true
		}
	})
	if !timeoutSet {
		timeout = secDuration(float64(interval) * 0.9)
	}

	hosts := flag.Args()
	if len(hosts) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	intervalDur := time.Duration(interval)
	timeoutDur := time.Duration(timeout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(intervalDur)
	defer ticker.Stop()

	t := time.Now()
	results := runCycle(ctx, hosts, timeoutDur)
	if ctx.Err() == nil {
		renderCycle(results, t, theme)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t := time.Now()
			results := runCycle(ctx, hosts, timeoutDur)
			if ctx.Err() == nil {
				renderCycle(results, t, theme)
			}
		}
	}
}
