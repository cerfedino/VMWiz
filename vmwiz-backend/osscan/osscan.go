// Package osscan probes a list of hosts on TCP/22 for an SSH banner and derives the OS codename from it.
// connect, read the first line, regex-match the version, map to a friendly Debian/Ubuntu codename.
package osscan

import (
	"bufio"
	"context"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// Status enumerates the outcome of a single host probe.
type Status string

const (
	StatusOK             Status = "ok"
	StatusUnreachable    Status = "unreachable"
	StatusNoSSHBanner    Status = "no_ssh_banner"
	StatusUnknownOS      Status = "unknown_os"
	StatusUnknownVersion Status = "unknown_version"
)

// Result captures everything we learned about one host.
type Result struct {
	Host        string `json:"host"`
	RDNS        string `json:"rdns,omitempty"` // reverse-DNS lookup result, when host was an IP
	Status      Status `json:"status"`
	Banner      string `json:"banner,omitempty"`
	OS          string `json:"os,omitempty"`          // "Debian" / "Ubuntu"
	Version     string `json:"version,omitempty"`     // e.g. "12"
	Codename    string `json:"codename,omitempty"`    // e.g. "Bookworm"
	PoolKey     string `json:"poolKey,omitempty"`     // stable group id, e.g. "debian-12"
	DisplayName string `json:"displayName,omitempty"` // pretty label, e.g. "Debian 12 (Bookworm)"
	Outdated    bool   `json:"outdated,omitempty"`
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"durationMs"`
}

// Pool groups hosts that share a poolKey (= same OS + version).
type Pool struct {
	PoolKey     string   `json:"poolKey"`
	OS          string   `json:"os"`
	Version     string   `json:"version"`
	Codename    string   `json:"codename"`
	DisplayName string   `json:"displayName"`
	Outdated    bool     `json:"outdated"`
	Hosts       []string `json:"hosts"`
}

// ScanReport bundles per-host results and the derived pools.
type ScanReport struct {
	Results []Result `json:"results"`
	Pools   []Pool   `json:"pools"`
}

var (
	debianVersionRE  = regexp.MustCompile(`deb(\d+)`)
	opensshVersionRE = regexp.MustCompile(`OpenSSH_(\d+\.\d+)`)
)

// debian codename, current?
var debianCodenames = map[string]struct {
	codename string
	current  bool
}{
	"14": {"Forky", true},
	"13": {"Trixie", true},
	"12": {"Bookworm", true},
	"11": {"Bullseye", false},
	"10": {"Buster", false},
	"9":  {"Stretch", false},
	"8":  {"Jessie", false},
}

// Debian's OpenSSH version per stable release. Used as a fallback when the
// banner does not carry a "deb<N>u<M>" suffix (e.g. backported builds or
// banners that strip the Debian-specific revision tag).
var debianByOpenSSH = map[string]string{
	"10.5": "14",
	"10.4": "14",
	"10.0": "13",
	"9.9":  "13",
	"9.2":  "12",
	"8.4":  "11",
	"7.9":  "10",
	"7.4":  "9",
	"6.7":  "8",
}

// ubuntu OpenSSH version -> release info
var ubuntuByOpenSSH = map[string]struct {
	version  string
	codename string
	current  bool
}{
	"10.2": {"26.04 LTS", "Resolute Raccoon", true},
	"10.0": {"25.10", "Questing Quokka", false},
	"9.9": {"25.04", "Plucky Puffin", false},
	"9.6": {"24.04 LTS", "Noble Numbat", true},
	"8.9": {"22.04 LTS", "Jammy Jellyfish", true},
	"8.2": {"20.04 LTS", "Focal Fossa", false},
	"7.6": {"18.04 LTS", "Bionic Beaver", false},
	"7.2": {"16.04 LTS", "Xenial Xerus", false},
}

// ScanOptions tunes the scanner. The zero value is safe.
type ScanOptions struct {
	Concurrency    int           // max parallel probes (default 32)
	ConnectTimeout time.Duration // dial + read deadline (default 3s)
}

func (o ScanOptions) withDefaults() ScanOptions {
	if o.Concurrency <= 0 {
		o.Concurrency = 32
	}
	if o.ConnectTimeout <= 0 {
		o.ConnectTimeout = 3 * time.Second
	}
	return o
}

// Scan probes every host concurrently and returns a complete report.
func Scan(hosts []string, opts ScanOptions) ScanReport {
	opts = opts.withDefaults()

	results := make([]Result, len(hosts))
	sem := make(chan struct{}, opts.Concurrency)
	var wg sync.WaitGroup

	for i, h := range hosts {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, host string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = probeHost(host, opts.ConnectTimeout)
		}(i, h)
	}
	wg.Wait()

	return ScanReport{
		Results: results,
		Pools:   groupIntoPools(results),
	}
}

func probeHost(host string, timeout time.Duration) Result {
	start := time.Now()
	res := Result{Host: host}

	// Best-effort reverse DNS so admins see a name when scanning IPs.
	if ip := net.ParseIP(host); ip != nil {
		resolver := &net.Resolver{}
		ctx, cancel := contextWithTimeout(timeout)
		defer cancel()
		if names, err := resolver.LookupAddr(ctx, host); err == nil && len(names) > 0 {
			res.RDNS = strings.TrimSuffix(names[0], ".")
		}
	}

	addr := net.JoinHostPort(host, "22")
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		res.Status = StatusUnreachable
		res.Error = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	banner, err := bufio.NewReader(conn).ReadString('\n')
	banner = strings.TrimSpace(banner)
	if err != nil && banner == "" {
		res.Status = StatusNoSSHBanner
		res.Error = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	res.Banner = banner

	classify(&res)
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

func classify(r *Result) {
	bannerLower := strings.ToLower(r.Banner)

	switch {
	case strings.Contains(bannerLower, "debian"):
		r.OS = "Debian"
		// Primary: parse the "deb<N>u<M>" tag (e.g. "deb12u7" -> 12).
		// Fallback: map the OpenSSH version to a Debian release (so a banner
		// without the deb-tag still classifies correctly).
		if m := debianVersionRE.FindStringSubmatch(bannerLower); len(m) >= 2 {
			r.Version = m[1]
		} else if m := opensshVersionRE.FindStringSubmatch(r.Banner); len(m) >= 2 {
			if v, ok := debianByOpenSSH[m[1]]; ok {
				r.Version = v
			}
		}
		if r.Version == "" {
			r.Status = StatusUnknownVersion
			r.PoolKey = "debian-unknown"
			r.DisplayName = "Debian (version unknown)"
			r.Outdated = true
			return
		}
		if info, ok := debianCodenames[r.Version]; ok {
			r.Codename = info.codename
			r.Outdated = !info.current
			r.DisplayName = "Debian " + r.Version + " (" + info.codename + ")"
		} else {
			r.Outdated = true
			r.DisplayName = "Debian " + r.Version + " (unknown codename)"
		}
		r.Status = StatusOK
		r.PoolKey = "debian-" + r.Version

	case strings.Contains(bannerLower, "ubuntu"):
		r.OS = "Ubuntu"
		m := opensshVersionRE.FindStringSubmatch(r.Banner)
		if len(m) < 2 {
			r.Status = StatusUnknownVersion
			r.PoolKey = "ubuntu-unknown"
			r.DisplayName = "Ubuntu (version unknown)"
			r.Outdated = true
			return
		}
		osshVer := m[1]
		if info, ok := ubuntuByOpenSSH[osshVer]; ok {
			r.Version = info.version
			r.Codename = info.codename
			r.Outdated = !info.current
			r.DisplayName = "Ubuntu " + info.version + " (" + info.codename + ")"
			r.Status = StatusOK
			r.PoolKey = "ubuntu-" + strings.ReplaceAll(info.version, " ", "_")
		} else {
			r.Status = StatusUnknownVersion
			r.PoolKey = "ubuntu-openssh-" + osshVer
			r.DisplayName = "Ubuntu (OpenSSH " + osshVer + ", unmapped)"
			r.Outdated = true
		}

	default:
		r.Status = StatusUnknownOS
		r.PoolKey = "other"
		r.DisplayName = "Unknown OS"
	}
}

func groupIntoPools(results []Result) []Pool {
	idx := map[string]*Pool{}
	for _, r := range results {
		if r.PoolKey == "" {
			continue
		}
		p, ok := idx[r.PoolKey]
		if !ok {
			p = &Pool{
				PoolKey:     r.PoolKey,
				OS:          r.OS,
				Version:     r.Version,
				Codename:    r.Codename,
				DisplayName: r.DisplayName,
				Outdated:    r.Outdated,
			}
			idx[r.PoolKey] = p
		}
		p.Hosts = append(p.Hosts, r.Host)
	}

	out := make([]Pool, 0, len(idx))
	for _, p := range idx {
		sort.Strings(p.Hosts)
		out = append(out, *p)
	}
	// Stable display order: outdated first (so admins notice), then by name.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Outdated != out[j].Outdated {
			return out[i].Outdated
		}
		return out[i].DisplayName < out[j].DisplayName
	})
	return out
}
