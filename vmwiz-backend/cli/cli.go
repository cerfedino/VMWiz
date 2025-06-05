package cli

import (
	"fmt"
	"os"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

type verb struct {
	name string
	desc string
	fun  func()
}

func check_health() {
	startupcheck.DoAllStartupChecks()
}

func list_requests() {
	all := false

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "all" || os.Args[i] == "--all" {
			all = true
			continue
		}

		fmt.Printf("unknown flag `%s`\n", os.Args[i])
		return
	}

	requests, err := storage.DB.GetAllVMRequests()

	if err != nil {
		fmt.Printf("failed to get VM requests: %v\n", err)
		return
	}

	numPrintedReqs := 0

	for _, req := range requests {
		if !all && req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}
		fmt.Printf("%s\n", req.ToString())
		numPrintedReqs += 1
	}

	if numPrintedReqs == 0 {
		fmt.Println("no requests to display.")
	}
}

func list_ips() {
	ips, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
	if err != nil {
		fmt.Printf("failed to fetch free IPs: %v\n", err)
		return
	}

	for i, ip := range *ips {
		fmt.Printf("%3d %v\n", i, ip.IP)
	}
}

func help() {
	for _, verb := range verbs {
		fmt.Printf("%-16s %s\n", verb.name, verb.desc)
	}
}

var verbs []verb

func initVerbs() {
	verbs = []verb{
		{
			name: "help",
			desc: "displays this help page",
			fun:  help,
		},
		{
			name: "check-health",
			desc: "checks if all services are running correctly",
			fun:  check_health,
		},
		{
			name: "list-requests",
			desc: "lists outstanding VM requests",
			fun:  list_requests,
		},
		{
			name: "list-ips",
			desc: "list available IPv4s",
			fun:  list_ips,
		},
	}
}

func Main() {
	initVerbs()

	if len(os.Args) < 2 {
		fmt.Println("you need to provide a verb to tell me what to do")
		help()
	} else {
		found := false
		for _, cmd := range verbs {
			if cmd.name == os.Args[1] {
				cmd.fun()
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("unknown verb `%s`\n", os.Args[1])
			help()
		}
	}
}
