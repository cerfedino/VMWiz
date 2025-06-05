package cli

import (
	"fmt"

	_ "embed"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"

	"github.com/avamsi/climate"
)

type vw struct{}

func (v *vw) Health() {
	startupcheck.DoAllStartupChecks()
}

type ip struct {
	V *vw
}

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

type request struct {
	V *vw
}

type listOptions struct {
	All bool // also display accepted and rejected requests
}

func (r *request) List(opts *listOptions) {
	requests, err := storage.DB.GetAllVMRequests()

	if err != nil {
		fmt.Printf("failed to get VM requests: %v\n", err)
		return
	}

	numPrintedReqs := 0

	for _, req := range requests {
		if opts.All && req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}
		fmt.Printf("%s\n", req.ToString())
		numPrintedReqs += 1
	}

	if numPrintedReqs == 0 {
		fmt.Println("no requests to display.")
	}
}

//go:generate go tool cligen md.cli
//go:embed md.cli
var md []byte

func Main() {
	p := climate.Struct[vw](climate.Struct[ip](), climate.Struct[request]())
	climate.RunAndExit(p, climate.WithMetadata(md))
}
