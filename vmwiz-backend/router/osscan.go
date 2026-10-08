package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/osscan"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/osscan/*

type osscanRunBody struct {
	IncludeProxmox  bool     `json:"includeProxmox"`
	IncludeSOSHosts bool     `json:"includeSosHosts"`
	CidrRanges      []string `json:"cidrRanges"`
	ExtraHosts      []string `json:"extraHosts"`
	TimeoutMs       int      `json:"timeoutMs"`
	Concurrency     int      `json:"concurrency"`
}

type osscanMailBody struct {
	Hosts             []string `json:"hosts"`
	Subject           string   `json:"subject"`
	Body              string   `json:"body"`
	AdditionalCc      []string `json:"additionalCc"`
	IncludeNoContact  bool     `json:"includeNoContact"`
	confirmationToken string
}

type osscanMailHostResult struct {
	Host       string   `json:"host"`
	Recipients []string `json:"recipients"`
	Sent       bool     `json:"sent"`
	Skipped    bool     `json:"skipped"`
	SkipReason string   `json:"skipReason,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type osscanMailResponse struct {
	Sent    int                    `json:"sent"`
	Skipped int                    `json:"skipped"`
	Failed  int                    `json:"failed"`
	PerHost []osscanMailHostResult `json:"perHost"`
}

// renderTemplate substitutes {{hostname}}-style placeholders in the
// admin-supplied template. Unknown placeholders are left as-is so admins
// can spot typos.
func renderTemplate(tmpl string, vars map[string]string) string {
	out := tmpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

// gatherProxmoxHosts returns the full list of cluster VM hostnames.
func gatherProxmoxHosts() ([]string, error) {
	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, vm := range *vms {
		if vm.Name == "" || vm.Template == 1 {
			continue
		}
		if _, dup := seen[vm.Name]; dup {
			continue
		}
		seen[vm.Name] = struct{}{}
		out = append(out, vm.Name)
	}
	sort.Strings(out)
	return out, nil
}

// vmContactsByHostname returns the contact emails for a VM by name. Returns
// nil + nil if no VM matches (caller can decide whether to flag).
func vmContactsByHostname(name string) ([]string, error) {
	vms, err := proxmox.GetAllClusterVMsByName(name)
	if err != nil {
		return nil, err
	}
	if vms == nil || len(*vms) == 0 {
		return nil, nil
	}
	contacts := []string(nil)
	for _, vm := range *vms {
		cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
		if err != nil {
			log.Printf("vmContactsByHostname: GetNodeVMConfig %s: %v", vm.Name, err)
			continue
		}
		contacts = proxmox.GetEmails(*cfg, contacts)
	}
	return contacts, nil
}

// findResultForHost is used when we need OS metadata for templating.
// Returns zero value if not found.
func findResultForHost(report osscan.ScanReport, host string) osscan.Result {
	for _, r := range report.Results {
		if r.Host == host {
			return r
		}
	}
	return osscan.Result{Host: host}
}

func addAllOsscanRoutes(r *mux.Router) {
	// Run a scan. Admin-only.
	r.Methods("POST").Path("/api/osscan/scan").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body osscanRunBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		hosts := []string{}
		seen := map[string]struct{}{}
		add := func(h string) {
			h = strings.TrimSpace(h)
			if h == "" {
				return
			}
			if _, dup := seen[h]; dup {
				return
			}
			seen[h] = struct{}{}
			hosts = append(hosts, h)
		}

		if body.IncludeProxmox {
			pmHosts, err := gatherProxmoxHosts()
			if err != nil {
				log.Printf("osscan: failed to gather proxmox hosts: %v", err)
				http.Error(w, "Failed to enumerate Proxmox VMs", http.StatusInternalServerError)
				return
			}
			for _, h := range pmHosts {
				add(h)
			}
		}
		if body.IncludeSOSHosts {
			sosHosts, err := storage.DB.GetSOSHosts(r.Context())
			if err != nil {
				log.Printf("osscan: failed to load SOS hosts: %v", err)
				http.Error(w, "Failed to load SOS hosts", http.StatusInternalServerError)
				return
			}
			for _, h := range sosHosts {
				add(h)
			}
		}
		for _, cidr := range body.CidrRanges {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			ips, err := osscan.ExpandCIDR(cidr)
			if err != nil {
				http.Error(w, fmt.Sprintf("CIDR error: %v", err), http.StatusBadRequest)
				return
			}
			for _, ip := range ips {
				add(ip)
			}
		}
		for _, h := range body.ExtraHosts {
			add(h)
		}

		if len(hosts) == 0 {
			http.Error(w, "No hosts to scan. Provide a CIDR range, enable SOS hosts, enable Proxmox VMs, or list extra hosts.", http.StatusBadRequest)
			return
		}

		opts := osscan.ScanOptions{
			Concurrency: body.Concurrency,
		}
		if body.TimeoutMs > 0 {
			opts.ConnectTimeout = time.Duration(body.TimeoutMs) * time.Millisecond
		}

		report := osscan.Scan(hosts, opts)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	})))

	// Send a templated mail to all VM owners in a pool. Admin-only.
	// Body fields:
	//   hosts:           list of hostnames in the pool
	//   subject, body:   template with {{hostname}}, {{os}}, {{version}}, {{codename}} placeholders
	//   additionalCc:    extra recipients added to every mail
	//   includeNoContact: when true, hosts with zero contacts are still tried (using only additionalCc)
	r.Methods("POST").Path("/api/osscan/mail").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body osscanMailBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if len(body.Hosts) == 0 {
			http.Error(w, "hosts must be non-empty", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Subject) == "" {
			http.Error(w, "subject is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Body) == "" {
			http.Error(w, "body is required", http.StatusBadRequest)
			return
		}

		// Re-scan only the requested hosts so we can populate template vars
		// (os/version/codename) without trusting client-supplied values.
		report := osscan.Scan(body.Hosts, osscan.ScanOptions{})

		resp := osscanMailResponse{PerHost: make([]osscanMailHostResult, 0, len(body.Hosts))}

		for _, host := range body.Hosts {
			scan := findResultForHost(report, host)
			contacts, err := vmContactsByHostname(host)
			if err != nil {
				resp.Failed++
				resp.PerHost = append(resp.PerHost, osscanMailHostResult{
					Host:  host,
					Error: fmt.Sprintf("contact lookup failed: %v", err),
				})
				continue
			}

			recipients := append([]string{}, contacts...)
			recipients = append(recipients, body.AdditionalCc...)
			recipients = dedupNonEmpty(recipients)

			if len(recipients) == 0 && !body.IncludeNoContact {
				resp.Skipped++
				resp.PerHost = append(resp.PerHost, osscanMailHostResult{
					Host:       host,
					Skipped:    true,
					SkipReason: "no contacts in VM description and includeNoContact=false",
				})
				continue
			}

			vars := map[string]string{
				"hostname": host,
				"os":       scan.OS,
				"version":  scan.Version,
				"codename": scan.Codename,
				"banner":   scan.Banner,
				"display":  scan.DisplayName,
			}
			subject := renderTemplate(body.Subject, vars)
			rendered := renderTemplate(body.Body, vars)

			if err := notifier.SendEmail(subject, []byte(rendered), recipients); err != nil {
				resp.Failed++
				resp.PerHost = append(resp.PerHost, osscanMailHostResult{
					Host:       host,
					Recipients: recipients,
					Error:      err.Error(),
				})
				continue
			}
			resp.Sent++
			resp.PerHost = append(resp.PerHost, osscanMailHostResult{
				Host:       host,
				Recipients: recipients,
				Sent:       true,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})))

	// Static info: defaults used to prefill the admin UI.
	r.Methods("GET").Path("/api/osscan/info").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"defaultCidr": osscan.DefaultCIDR,
		})
	})))

	// Read the current SOS host list.
	r.Methods("GET").Path("/api/osscan/soshosts").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hosts, err := storage.DB.GetSOSHosts(r.Context())
		if err != nil {
			log.Printf("osscan: failed to load SOS hosts: %v", err)
			http.Error(w, "Failed to load SOS hosts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"hosts": hosts})
	})))

	// Replace the SOS host list. Body: { hosts: string[] }.
	r.Methods("POST").Path("/api/osscan/soshosts").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Hosts []string `json:"hosts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		hosts, err := storage.DB.SetSOSHosts(r.Context(), body.Hosts)
		if err != nil {
			log.Printf("osscan: failed to save SOS hosts: %v", err)
			http.Error(w, "Failed to save SOS hosts", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"hosts": hosts,
		})
	})))

	// Preview which hosts would receive mail and from which contacts. No
	// mail is sent. Useful for the admin UI to show counts before firing.
	r.Methods("POST").Path("/api/osscan/mail/preview").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body osscanMailBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		type previewEntry struct {
			Host       string   `json:"host"`
			Recipients []string `json:"recipients"`
			Skipped    bool     `json:"skipped"`
			SkipReason string   `json:"skipReason,omitempty"`
		}
		out := []previewEntry{}
		totalMails := 0
		for _, host := range body.Hosts {
			contacts, err := vmContactsByHostname(host)
			if err != nil {
				out = append(out, previewEntry{Host: host, Skipped: true, SkipReason: err.Error()})
				continue
			}
			recipients := append([]string{}, contacts...)
			recipients = append(recipients, body.AdditionalCc...)
			recipients = dedupNonEmpty(recipients)
			if len(recipients) == 0 && !body.IncludeNoContact {
				out = append(out, previewEntry{Host: host, Skipped: true, SkipReason: "no contacts"})
				continue
			}
			totalMails++
			out = append(out, previewEntry{Host: host, Recipients: recipients})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"totalMails": totalMails,
			"perHost":    out,
		})
	})))
}

func dedupNonEmpty(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
