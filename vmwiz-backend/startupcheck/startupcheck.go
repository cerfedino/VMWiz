package startupcheck

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"slices"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/fatih/color"
)

type StartupCheck struct {
	Name      string
	Successes []string
	Errors    []error
	Warnings  []string
}

var err_indicator = color.RedString("X")
var warn_indicator = color.YellowString("!")
var succ_indicator = color.GreenString("+")

func (s *StartupCheck) AddSuccess(success string) {
	s.Successes = append(s.Successes, success)
}

func (s *StartupCheck) AddError(err error) {
	s.Errors = append(s.Errors, err)
}

func (s *StartupCheck) AddWarning(warning string) {
	s.Warnings = append(s.Warnings, warning)
}

func (s *StartupCheck) String() string {
	var ret string = ""

	var indicator string = succ_indicator
	if len(s.Errors) > 0 {
		indicator = err_indicator
	} else if len(s.Warnings) > 0 {
		indicator = warn_indicator
	}
	ret += fmt.Sprintf("[%v] %v\n", indicator, s.Name)

	for _, succ := range s.Successes {
		ret += fmt.Sprintf("\t[%v] %s\n", succ_indicator, succ)
	}
	for _, warn := range s.Warnings {
		ret += fmt.Sprintf("\t[%v] %v\n", warn_indicator, warn)
	}
	for _, err := range s.Errors {
		ret += fmt.Sprintf("\t[%v] %v\n", err_indicator, err)
	}

	return ret
}

func DoAllStartupChecks() bool {
	fatal := false

	var startupChecks []*StartupCheck
	startupChecks = slices.Concat(DoDatabaseStartupChecks(), DoNetcenterStartupChecks(), DoProxmoxStartupChecks())
	for _, check := range startupChecks {
		log.Println((*check).String())
		if len((*check).Errors) > 0 {
			fatal = true
		}
	}

	return fatal
}

func DoNetcenterStartupChecks() []*StartupCheck {
	// Check that env variables are not empty
	var checks []*StartupCheck

	env_check := StartupCheck{
		Name: "Netcenter Environment Variables",
	}

	checks = append(checks, &env_check)
	required_env := []string{"NETCENTER_HOST", "NETCENTER_USER", "NETCENTER_PWD"}
	for _, env := range required_env {
		if os.Getenv(env) == "" {
			env_check.AddError(fmt.Errorf("%v is not set", env))
		} else {
			env_check.AddSuccess(fmt.Sprintf("%v is set", env))
		}
	}
	if len(env_check.Errors) > 0 {
		return checks
	}

	// Make sure Netcenter is reachable
	query_check := StartupCheck{
		Name: "Run example Netcenter query",
	}
	checks = append(checks, &query_check)
	_, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
	if err != nil {
		query_check.AddError(fmt.Errorf("Error when performing HTTP request to Netcenter:  %v", err.Error()))
	} else {
		query_check.AddSuccess("Successfully executed query")
	}
	if len(query_check.Errors) > 0 {
		return checks
	}

	// Check how many free IPs are there
	ip_availability_check := StartupCheck{
		Name: "Check availability of free IPs",
	}
	checks = append(checks, &ip_availability_check)
	ipv4s, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
	if err != nil {
		ip_availability_check.AddError(fmt.Errorf("Error when checking availability of free IPv4s: %v", err.Error()))
	} else {
		msg := fmt.Sprintf("Found %d free IPv4 addresses in subnet '%v'", len(*ipv4s), netcenter.VM_SUBNET.V4net)
		if len(*ipv4s) < 20 {
			ip_availability_check.AddWarning(msg)
		} else {
			ip_availability_check.AddSuccess(msg)
		}
	}
	ipv6, err := netcenter.GetFreeIPv6sInSubnet(netcenter.VM_SUBNET.V6net)
	if err != nil {
		ip_availability_check.AddError(fmt.Errorf("Error when checking availability of free IPv6s: %v", err.Error()))
	} else {
		msg := fmt.Sprintf("Found %d free IPv6 addresses in subnet '%v'", len(*ipv6), netcenter.VM_SUBNET.V6net)
		if len(*ipv6) < 20 {
			ip_availability_check.AddWarning(msg)
		} else {
			ip_availability_check.AddSuccess(msg)
		}
	}

	return checks
}

func DoProxmoxStartupChecks() []*StartupCheck {
	// Check that env variables are not empty
	var checks []*StartupCheck

	env_check := StartupCheck{
		Name: "Proxmox Environment Variables",
	}
	checks = append(checks, &env_check)
	required_pve_env := []string{"PVE_HOST", "PVE_USER", "PVE_TOKENID", "PVE_UUID"}
	for _, env := range required_pve_env {
		if os.Getenv(env) == "" {
			env_check.AddError(fmt.Errorf("%v is not set", env))
		} else {
			env_check.AddSuccess(fmt.Sprintf("%v is set", env))
		}
	}
	if len(env_check.Errors) > 0 {
		return checks
	}

	ping_check := StartupCheck{
		Name: "PVE ping test",
	}
	checks = append(checks, &ping_check)
	pve_url, err := url.Parse(os.Getenv("PVE_HOST"))
	if err != nil {
		ping_check.AddError(fmt.Errorf("Couldn't parse PVE_HOST: %v", err.Error()))
	}
	cmd := exec.Command("ping", "-c", "3", pve_url.Hostname())
	_, err = cmd.Output()
	if err != nil {
		ping_check.AddError(fmt.Errorf("Unsuccessful ping of %v: %v", pve_url.Hostname(), err.Error()))
	} else {
		ping_check.AddSuccess("Successfully pinged " + pve_url.Hostname())
	}
	if len(ping_check.Errors) > 0 {
		return checks
	}

	access_check := StartupCheck{
		Name: "PVE HTTP API authentication test",
	}
	checks = append(checks, &access_check)
	_, err = proxmox.GetTokenPermissions()
	if err != nil {
		access_check.AddError(fmt.Errorf("Failed to get permissions: %v", err.Error()))
	} else {
		access_check.AddSuccess("Successfully authenticated with PVE HTTP API")
	}

	// TODO: Check whether token has enough permissions to request all endpoints in codebase

	ssh_env_check := StartupCheck{
		Name: "Cluster Manager SSH environment variables",
	}
	checks = append(checks, &ssh_env_check)
	required_cm_env := []string{"SSH_CM_HOST", "SSH_CM_USER"}
	for _, env := range required_cm_env {
		if os.Getenv(env) == "" {
			ssh_env_check.AddError(fmt.Errorf("%v is not set", env))
		} else {
			ssh_env_check.AddSuccess(fmt.Sprintf("%v is set", env))
		}
	}
	if len(env_check.Errors) > 0 {
		return checks
	}

	ssh_access_check := StartupCheck{
		Name: "Cluster Manager SSH authentication test",
	}
	checks = append(checks, &ssh_access_check)
	err = proxmox.TestCMConnection()
	if err != nil {
		ssh_access_check.AddError(fmt.Errorf("Failed to estabilish SSH connection with Cluster Manager: %v", err.Error()))
	} else {
		ssh_access_check.AddSuccess("Successfully authenticated with Cluster Manager SSH")
	}

	return checks
}

func DoDatabaseStartupChecks() []*StartupCheck {
	var checks []*StartupCheck

	env_check := StartupCheck{
		Name: "PostgreSQL Environment Variables",
	}
	checks = append(checks, &env_check)
	required_pve_env := []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
	for _, env := range required_pve_env {
		if os.Getenv(env) == "" {
			env_check.AddError(fmt.Errorf("%v is not set", env))
		} else {
			env_check.AddSuccess(fmt.Sprintf("%v is set", env))
		}
	}
	if len(env_check.Errors) > 0 {
		return checks
	}

	connection_check := StartupCheck{
		Name: "Testing database connection",
	}
	if err := storage.DB.CreateConnection(); err != nil {
		connection_check.AddError(fmt.Errorf("Couldn't connect to DB : %v", err.Error()))
	} else {
		connection_check.AddSuccess("Connection to DB successful")
	}
	checks = append(checks, &connection_check)

	migration_check := StartupCheck{
		Name: "Testing database migration",
	}
	if err := storage.DB.InitMigrations(); err != nil {
		migration_check.AddError(fmt.Errorf("Couldn't create migrations: %v", err.Error()))
	} else {
		migration_check.AddSuccess("Migrations created successfully")
	}
	checks = append(checks, &migration_check)

	return checks
}
