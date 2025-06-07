package main

import (
	"fmt"
	"log"
	"os"

	_ "embed"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"

	"github.com/avamsi/climate"
)

type vw struct {
}

// check if the environment is properly set up and the other services are reachable
func (v *vw) Health() {
	startupcheck.DoAllStartupChecks()
}

// look at IPs
type ip struct {
	V *vw
}

// free IPv4s
func (i *ip) List() {
	ips, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
	if err != nil {
		fmt.Printf("failed to fetch free IPs: %v\n", err)
		return
	}

	for i, ip := range *ips {
		fmt.Printf("%3d %v\n", i, ip.IP)
	}
}

// look at or process requests
type request struct {
	V *vw
}

type listOptions struct {
	All bool // also display accepted and rejected requests
}

// display VM requests in database
func (r *request) List(opts *listOptions) {
	requests, err := storage.DB.GetAllVMRequests()

	if err != nil {
		fmt.Printf("failed to get VM requests: %v\n", err)
		return
	}

	numPrintedReqs := 0

	for _, req := range requests {
		if !opts.All && req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}
		fmt.Printf("%s\n", req.ToString())
		numPrintedReqs += 1
	}

	if numPrintedReqs == 0 {
		fmt.Println("no requests to display.")
	}
}

type acceptOrRejectOptions struct {
	Id   int    `default:"-1"` // ID to accept or reject
	Name string // host to accept or reject
}

func (opts *acceptOrRejectOptions) find() int64 {
	if opts.Id >= 0 {
		return int64(opts.Id)
	}

	requests, err := storage.DB.GetAllVMRequests()

	if err != nil {
		fmt.Printf("failed to get VM requests: %v\n", err)
		os.Exit(-1)
	}

	for _, req := range requests {
		if req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}

		if req.ToVMOptions().FQDN == opts.Name {
			return req.ID
		}
	}

	for _, req := range requests {
		if req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}

		if req.ToVMOptions().FQDN == fmt.Sprintf("%s.vsos.ethz.ch", opts.Name) {
			return req.ID
		}

		if req.ToVMOptions().FQDN == fmt.Sprintf("%s.sos.ethz.ch", opts.Name) {
			return req.ID
		}
	}

	fmt.Printf("Did not recognize host name %s\n", opts.Name)
	os.Exit(-1)

	return 0
}

// accept the given request
func (r *request) Accept(opts *acceptOrRejectOptions) {
	id := opts.find()
	router.AcceptVMRequest(id)
}

// reject the given request
func (r *request) Reject(opts *acceptOrRejectOptions) {
	id := opts.find()
	router.RejectVMRequest(id)
}

// get information about surveys
type survey struct {
}

func listSurveyHosts(fun func(int) ([]string, error)) {
	id, err := storage.DB.SurveyGetLastId()
	if err != nil {
		fmt.Printf("error fetching last survey id: %v\n", err)
		os.Exit(-1)
	}

	hosts, err := fun(id)
	if err != nil {
		fmt.Printf("error fetching survey data from DB: %v\n", err)
	}

	for _, host := range hosts {
		fmt.Println(host)
	}
}

// list the VMs where people said they still need it
func (s *survey) ListPositive() {
	listSurveyHosts(storage.DB.SurveyEmailPositive)
}

// list the VMs where people said they can be deleted
func (s *survey) ListNegative() {
	listSurveyHosts(storage.DB.SurveyEmailNegative)
}

// list the VMs where people did not yet answer
func (s *survey) ListUnanswered() {
	listSurveyHosts(storage.DB.SurveyEmailNotResponded)
}

func sliceToSet[T comparable](slice []T) map[T]struct{} {
	set := make(map[T]struct{})
	for _, val := range slice {
		set[val] = struct{}{}
	}
	return set
}

func (s *survey) ShutdownUnanswered() {
	id, err := storage.DB.SurveyGetLastId()
	if err != nil {
		fmt.Printf("error fetching last survey id: %v\n", err)
		os.Exit(-1)
	}

	shutdownList, err := storage.DB.SurveyEmailNotResponded(id)
	if err != nil {
		fmt.Printf("error fetching survey data from DB: %v\n", err)
		os.Exit(-1)
	}

	shutdownSet := sliceToSet(shutdownList)

	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		fmt.Printf("error fetching cluster VMs")
		os.Exit(-1)
	}

	hasErrors := false

	for _, vm := range *vms {
		_, doShutdown := shutdownSet[vm.Name]
		if !doShutdown {
			continue
		}

		fmt.Printf("shutting down %s...\n", vm.Name)

		err = proxmox.ShutdownVMWithReason(vm.Node, vm.Vmid, "the owner did not respond to the survey.")
		if err != nil {
			fmt.Printf("Failed to shut down VM %s: %v", vm.Name, err)
			hasErrors = true
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

//go:generate go tool cligen md.cli
//go:embed md.cli
var md []byte

func main() {
	err := config.AppConfig.Init()
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err.Error())
	}

	err = storage.DB.Init()
	if err != nil {
		log.Fatalf("Error on startup: %v", err.Error())
	}

	p := climate.Struct[vw](climate.Struct[ip](), climate.Struct[request](), climate.Struct[survey]())
	climate.RunAndExit(p, climate.WithMetadata(md))
}
