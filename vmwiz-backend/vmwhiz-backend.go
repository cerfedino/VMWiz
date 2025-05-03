package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/rs/cors"
	"golang.org/x/exp/rand"
)

// func (s *StartupChecks) String() string {

// 	ret := log.Println("[-] %v\n", s.Name)
// 	for _, result := range s.Results {
// 		err, ok := result.(error)
// 		if ok {
// 			ret += fmt.Sprintf("\t[ERROR] %v\n", err.Error())
// 			continue
// 		}

// 		if reflect.TypeOf(err) == reflect.TypeOf(errors.New("")) {
// 			ret += fmt.Sprintf("\t[ERROR] %v\n", err.Error())
// 		} else {
// 			ret += fmt.Sprintf("\t[OK] %v\n", err)
// 		}

// 	}
// 	return ret
// }

func main() {
	// TODO: Remove in prod
	// rand.Seed(uint64((time.Now().UnixNano())))
	rand.Seed(uint64(42))

	err := config.AppConfig.Init()
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err.Error())
	}

	notifier.InitSMTP()

	if startupcheck.DoAllStartupChecks() {
		log.Fatalf("Startup checks failed. Exiting ...")
	} else {
		log.Println("Startup checks passed.")
	}

	err = storage.DB.Init()
	if err != nil {
		log.Fatalf("Error on startup: %v", err.Error())
	}

	auth.Init()

	nodes, err := proxmox.GetAllNodeVMsByName("comp-epyc-lee-3", "vmwiz-test.vsos.ethz.ch")
	if err != nil {
		log.Println(err)
	} else {
		if len(*nodes) > 0 {
			err = proxmox.ForceStopNodeVM("testing", "comp-epyc-lee-3", (*nodes)[0].Vmid)
			if err != nil {
				log.Println(err)
			}
			err = proxmox.DeleteNodeVM("testing", "comp-epyc-lee-3", (*nodes)[0].Vmid, true, true, false)
			if err != nil {
				log.Println(err)
			}
		}
	}

	err = netcenter.DeleteDNSEntryByHostname("testing", "vmwiz-test.vsos.ethz.ch")
	if err != nil {
		log.Println(err)
	}

	// _, _, err = proxmox.CreateVM(proxmox.VMCreationOptions{
	// 	Template:     proxmox.IMAGE_UBUNTU_24_04,
	// 	FQDN:         "vmwiz-test.vsos.ethz.ch",
	// 	Reinstall:    false,
	// 	Cores_CPU:    5,
	// 	Tags:         []string{"created-by-vmwiz"},
	// 	RAM_MB:       1024,
	// 	Disk_GB:      10,
	// 	UseQemuAgent: true,
	// 	Notes:        "Test VM",
	// 	SSHPubkeys:   []string{"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBO1IgyOIr5Sx9/Re60E4A6D2KLX9sT8bLl/8mKpS0P8O0wTj82T6/qPWJWeuOfOYP5bj0yErK0Y1xgiTVOePgws= cerfe@sirius"},
	// })
	// if err != nil {
	// 	log.Println(err)
	// }
	// log.Println((*vm).Tags)

	cors := cors.New(cors.Options{
		// Allowing the Vue frontend to access the API
		AllowedOrigins:   []string{"vmwiz-frontend"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Handler:      cors.Handler(router.Router()),
		Addr:         ":8081",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Printf("Listening on %s ...\n", srv.Addr)
		srv.ListenAndServe()
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Shutting down ...")
	os.Exit(0)
}
