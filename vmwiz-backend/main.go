package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/server"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/urfave/cli/v3"
)

func main() {
	err := config.AppConfig.Init()
	if err != nil {
		log.Printf("Failed to parse config: %v", err.Error())
		return
	}

	notifier.InitSMTP()

	err = storage.DB.Init()
	if err != nil {
		log.Printf("Error on startup: %v", err.Error())
		return
	}

	auth.Init()
	confirmation.Init()

	cmd := &cli.Command{
		Name:                  "vmwiz-backend",
		Usage:                 "CLI tool for managing VMWiz backend",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "server",
				Aliases:     []string{},
				Description: "Starts the VMWiz backend server",
				Action:      handle_survey,
			},
			{
				Name:        "health",
				Aliases:     []string{},
				Description: "Check if the environment is properly set up and the other services are reachable",
				Action:      handle_health,
			},
			{
				Name:        "ip",
				Aliases:     []string{},
				Description: "Manage IP addresses",
				Commands: []*cli.Command{
					{
						Name:        "list",
						Description: "list all free IPv4 addresses",
						Action:      handle_ip_list,
					},
				},
			},
			{
				Name:        "request",
				Description: "look at or process requests",
				Commands: []*cli.Command{
					{
						Name:        "list",
						Description: "list all free IPv4 addresses",
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "all",
								Usage: "also display accepted and rejected requests",
								Value: false,
							},
						},
						Action: handle_request_list,
					},
					{
						Name:        "accept",
						Description: "accept a VM request",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:  "id",
								Usage: "ID of the VM request",
								Value: -1,
							},
							&cli.StringFlag{
								Name:  "name",
								Usage: "Hostname of the VM request (e.g myvm.vsos.ethz.ch)",
								Value: "",
							},
						},
						Action: handle_request_accept,
					},
					{
						Name:        "reject",
						Description: "reject a VM request by ID",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:  "id",
								Usage: "ID of the VM request",
								Value: -1,
							},
							&cli.StringFlag{
								Name:  "name",
								Usage: "Hostname of the VM request (e.g myvm.vsos.ethz.ch)",
								Value: "",
							},
						},
						Action: handle_request_reject,
					},
				},
			},
			{
				Name:        "survey",
				Description: "look at or process VM usage surveys",
				Commands: []*cli.Command{
					{
						Name:        "list",
						Description: "list all surveys",
						Action:      handle_survey_list,
					},
					{
						Name:        "inspect",
						Description: "inspect the details of a survey",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:     "id",
								Usage:    "ID of the survey to inspect",
								Required: true,
							},
							&cli.BoolFlag{
								Name:  "positives",
								Usage: "List VMs that were reported as still in use",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "negatives",
								Usage: "List VMs that were reported as no longer needed",
								Value: false,
							},
							&cli.BoolFlag{
								Name:  "unanswered",
								Usage: "List VMs that were not answered in the survey",
								Value: false,
							},
						},
						Action: handle_survey_inspect,
					},
					{
						Name:        "shutdownunanswered",
						Description: "shutdown VMs that did not respond to the survey",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:     "id",
								Usage:    "ID of the survey",
								Required: true,
							},
						},
						Action: handle_survey_shutdownunanswered,
					},
				},
			},
			{
				Name:        "sanity",
				Description: "perform checks on the whole cluster and report potentially dangerous configurations",
				Action:      handle_sanity,
			},
			{
				Name:        "emails",
				Description: "get a list of all e-mail addresses",
				Action:      handle_emails,
			},

			{
				Name:        "descriptions",
				Description: "get all VM descriptions",
				Action:      handle_descriptions,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// helper function that looks up a VM request by ID or hostname, and checks that it is pending if justpending is true. If multiple matching requests are found, an error is returned.
func findVMRequest(id int, name string, justpending bool) (*storage.SQLVMRequest, error) {
	vmrequests := []*storage.SQLVMRequest{}
	if id >= 0 {
		vmrequest, err := storage.DB.GetVMRequestById(int64(id))
		if err != nil {
			return nil, err
		}
		if vmrequest != nil {
			vmrequests = append(vmrequests, vmrequest)
		}

	} else {
		reqs, err := storage.DB.GetVMRequestByHostname(name)
		if err != nil {
			return nil, err
		}
		vmrequests = append(vmrequests, reqs...)
	}

	res := []storage.SQLVMRequest{}
	for _, req := range vmrequests {
		if justpending && req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}
		res = append(res, *req)
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("No pending VM request found with hostname %s", name)

	} else if len(res) > 1 {
		var sb strings.Builder
		for _, req := range res {
			sb.WriteString(req.ToString() + "\n")
		}
		return nil, fmt.Errorf("Multiple pending VM requests found with hostname %s, please specify the ID. Matching requests:\n%s", name, sb.String())
	}

	return &res[0], nil
}

func sliceToSet[T comparable](slice []T) map[T]struct{} {
	set := make(map[T]struct{})
	for _, val := range slice {
		set[val] = struct{}{}
	}
	return set
}

func handle_survey(ctx context.Context, cmd *cli.Command) error {
	server.StartServer()
	return nil
}
func handle_health(ctx context.Context, cmd *cli.Command) error {
	startupcheck.DoAllStartupChecks()
	return nil
}
func handle_ip_list(ctx context.Context, cmd *cli.Command) error {
	ips, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
	if err != nil {
		fmt.Printf("failed to fetch free IPs: %v\n", err)
		return err
	}

	for i, ip := range *ips {
		fmt.Printf("%3d %v\n", i, ip.IP)
	}
	return nil
}
func handle_request_list(ctx context.Context, cmd *cli.Command) error {
	requests, err := storage.DB.GetAllVMRequests()
	if err != nil {
		return err
	}

	numPrintedReqs := 0

	for _, req := range requests {
		if !cmd.Bool("all") && req.RequestStatus != storage.REQUEST_STATUS_PENDING {
			continue
		}
		fmt.Printf("%s\n", req.ToString())
		numPrintedReqs += 1
	}

	if numPrintedReqs == 0 {
		fmt.Println("no requests to display.")
	}
	return nil
}
func handle_request_accept(ctx context.Context, cmd *cli.Command) error {
	if cmd.Int("id") == -1 && cmd.String("name") == "" {
		return fmt.Errorf("Either --id or --name must be provided")
	}
	vmrequest, err := findVMRequest(cmd.Int("id"), cmd.String("name"), true)
	if err != nil {
		return err
	}
	fmt.Printf("Accepting VM request:\n%s\n", vmrequest.ToString())

	fmt.Println("Confirm? (y/n): ")
	var response string
	fmt.Scan(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	errB := router.AcceptVMRequest(vmrequest.ID)
	if errB != nil {
		return fmt.Errorf("%s: %v\n", errB.Err, errB.UserMsg)
	}

	return nil
}
func handle_request_reject(ctx context.Context, cmd *cli.Command) error {
	if cmd.Int("id") == -1 && cmd.String("name") == "" {
		return fmt.Errorf("Either --id or --name must be provided")
	}
	vmrequest, err := findVMRequest(cmd.Int("id"), cmd.String("name"), true)
	if err != nil {
		return err
	}
	fmt.Printf("Rejecting VM request:\n%s\n", vmrequest.ToString())

	fmt.Println("Confirm? (y/n): ")
	var response string
	fmt.Scan(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	errB := router.RejectVMRequest(vmrequest.ID)
	if errB != nil {
		return fmt.Errorf("%s: %v\n", errB.Err, errB.UserMsg)
	}
	return nil
}
func handle_survey_list(ctx context.Context, cmd *cli.Command) error {
	surveys, err := storage.DB.SurveyGetAll()
	if err != nil {
		return fmt.Errorf("failed to get surveys: %v", err)
	}

	for _, survey := range surveys {
		fmt.Printf("%s\n", survey.ToString())
	}
	return nil
}
func handle_survey_inspect(ctx context.Context, cmd *cli.Command) error {
	surveyId := cmd.Int("id")
	positives := cmd.Bool("positives")
	negatives := cmd.Bool("negatives")
	unanswered := cmd.Bool("unanswered")

	positiveList := []string{}
	negativeList := []string{}
	unansweredList := []string{}

	negativeCount, err := storage.DB.SurveyEmailCountNegative(int64(surveyId))
	if err != nil {
		return err
	}
	positiveCount, err := storage.DB.SurveyEmailCountPositive(int64(surveyId))
	if err != nil {
		return err
	}
	unansweredCount, err := storage.DB.SurveyEmailCountNotResponded(int64(surveyId))
	if err != nil {
		return err
	}

	if positives {
		positiveList, err = storage.DB.SurveyEmailPositive(surveyId)
		if err != nil {
			return err
		}
	}
	if negatives {
		negativeList, err = storage.DB.SurveyEmailNegative(surveyId)
		if err != nil {
			return err
		}
	}
	if unanswered {
		unansweredList, err = storage.DB.SurveyEmailNotResponded(surveyId)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Survey ID %d:\n", surveyId)
	fmt.Printf("Still in use: %d\n", *positiveCount)
	fmt.Printf("No longer needed: %d\n", *negativeCount)
	fmt.Printf("Unanswered: %d\n", *unansweredCount)

	if positives {
		fmt.Printf("\nStill in use:\n\t%s\n", strings.Join(positiveList, "\n\t"))
	}
	if negatives {
		fmt.Printf("\nNo longer needed:\n\t%s\n", strings.Join(negativeList, "\n\t"))
	}
	if unanswered {
		fmt.Printf("\nUnanswered:\n\t%s\n", strings.Join(unansweredList, "\n\t"))
	}
	return nil
}
func handle_survey_shutdownunanswered(ctx context.Context, cmd *cli.Command) error {
	surveyId := cmd.Int("id")
	shutdownList, err := storage.DB.SurveyEmailNotResponded(surveyId)
	if err != nil {
		return err
	}

	shutdownSet := sliceToSet(shutdownList)

	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return err
	}

	fmt.Printf("Shutting down the following VMs:\n%s\n\nConfirm? (y/n): ", strings.Join(shutdownList, "\n"))
	var response string
	fmt.Scan(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	errors := []string{}

	for _, vm := range *vms {
		_, doShutdown := shutdownSet[vm.Name]
		if !doShutdown {
			continue
		}

		fmt.Printf("Shutting down %s...\n", vm.Name)

		err = proxmox.ShutdownVMWithReason(vm.Node, vm.Vmid, "the owner did not respond to the survey.")
		if err != nil {
			fmt.Printf("Failed to shut down VM %s: %v", vm.Name, err)
			errors = append(errors, fmt.Sprintf("Failed to shut down VM %s: %v", vm.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("Errors occurred during shutdown:\n%s", strings.Join(errors, "\n"))
	}
	return nil
}

func handle_sanity(ctx context.Context, cmd *cli.Command) error {
	warns := proxmox.CheckAllVMs()

	if len(warns) == 0 {
		fmt.Println("Everything seems healthy.")
	} else {
		for i, w := range warns {
			fmt.Printf("%4d  %-30s  %-15s  %-40s\n", i, w.VM.Name, w.Category, w.Detail)
		}
	}
	return nil
}

func handle_emails(ctx context.Context, cmd *cli.Command) error {
	vms, err := proxmox.GetAllClusterVMs()
	if err != nil {
		return err
	}
	emails := []string(nil)
	errors := []string{}
	for _, vm := range *vms {
		desc, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to get VM config for %s: %v", vm.Name, err))
			continue
		}
		emails = proxmox.GetEmails(*desc, emails)
	}

	for _, email := range emails {
		fmt.Println(email)
	}

	if len(errors) > 0 {
		return fmt.Errorf("Some errors occurred while fetching VM configs:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

func handle_descriptions(ctx context.Context, cmd *cli.Command) error {
	vms, _ := proxmox.GetAllClusterVMs()
	errors := []string{}
	for _, vm := range *vms {
		desc, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to get VM config for %s: %v", vm.Name, err))
			continue
		}
		fmt.Printf("%s\n%s\n\n", vm.Name, desc.Description)
	}

	if len(errors) > 0 {
		return fmt.Errorf("Some errors occurred while fetching VM configs:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
