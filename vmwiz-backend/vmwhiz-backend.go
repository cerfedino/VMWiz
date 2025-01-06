package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/rs/cors"
)

// type StartupChecks struct {
// 	Name   string
// 	Errors []any
// }

// func (s *StartupChecks) String() string {
// 	ret := "[-] Checks for " + s.Name + ":\n"
// 	for _, err := range s.Errors {
// 		if reflect.TypeOf(err) {
// 			ret += fmt.Sprintf("\t[ERROR] %v\n", err)
// 		} else {
// 			ret += fmt.Sprintf("\t[OK] %v\n", err)
// 		}

// 	}
// 	return ret
// }

// func DoChecks() []error {
// 	return []error{}
// }

func main() {
	storage.DB.Init("")

	// TODO: Remove. testing purposes only
	chosenIP, err := netcenter.Registerhost("vm", "vmwiz-test.vsos.ethz.ch")
	if err != nil {
		log.Println(err)
		return
	}

	// ? Deleting the DNS entry
	err = netcenter.DeleteDNSEntryByIP(chosenIP)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
	}
	fmt.Println("Host deleted !")

	// fmt.Println(proxmox.IsHostnameTaken(""))
	// err := proxmox.CreateVM(proxmox.PVEVMOptions{
	// 	Template:     "noble",
	// 	FQDN:         "cerfedinoo.vsos.ethz.ch",
	// 	Reinstall:    false,
	// 	RAM_MB:       1024,
	// 	Disk_GB:      10,
	// 	UseQemuAgent: true,
	// 	Description:  "Test VM",
	// 	SSHKeys:      []string{"ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBAAjFvUZr/m8zoXKW5wjNBXehNO9u7oiS+VchueNGA7Fa05aeI7KaP5iEDRUJ9fvfqOprV3z7OAv11lrJ0IKcsLOFQErfl1IrmErot0UJ6sDbAAmnKbr9gjqA0qQcDSNNKRjj7BkKd7zQGvOEjy179q9mvcNNMINFrPXjk2qvIBFHg1hnQ== cerfe@student-net-nw-0407.intern.ethz.ch"},
	// })
	// if err != nil {
	// 	log.Println(err)
	// }

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
	signal.Notify(c, os.Interrupt, syscall.SIGKILL)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 5000)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Shutting down ...")
	os.Exit(0)
}
