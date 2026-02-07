package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"github.com/rs/cors"
)

func StartServer() error {

	if startupcheck.DoAllStartupChecks() {
		log.Println("Startup checks failed")
		return fmt.Errorf("Startup checks failed")
	} else {
		log.Println("Startup checks passed.")
	}

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

	// nodes, err := proxmox.GetAllNodeVMsByName("comp-epyc-lee-3", "vmwiz-test.vsos.ethz.ch")
	// if err != nil {
	// 	log.Println(err)
	// } else {
	// 	if len(*nodes) > 0 {
	// 		err = proxmox.ForceStopNodeVM("comp-epyc-lee-3", (*nodes)[0].Vmid)
	// 		if err != nil {
	// 			log.Println(err)
	// 		}
	// 		err = proxmox.DeleteNodeVM("comp-epyc-lee-3", (*nodes)[0].Vmid, true, true, false)
	// 		if err != nil {
	// 			log.Println(err)
	// 		}
	// 	}
	// }

	// err = netcenter.DeleteDNSEntryByHostname("vmwiz-test.vsos.ethz.ch")
	// if err != nil {
	// 	log.Println(err)
	// }

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

	// for i := 0; i < 100; i += 1 {
	// 	fmt.Print(i, " ")
	// 	token, err := confirmation.NewToken()
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 	} else {
	// 		fmt.Println(*token)
	// 	}
	// 	time.Sleep(500)
	// }

	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Shutting down ...")
	return nil
}
