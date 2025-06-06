package main

import (
	"fmt"
	"log"
	"os"

	_ "embed"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
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

	p := climate.Struct[vw](climate.Struct[ip](), climate.Struct[request]())
	climate.RunAndExit(p, climate.WithMetadata(md))
}
