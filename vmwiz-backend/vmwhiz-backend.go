package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/rs/cors"
)

// func (s *StartupChecks) String() string {

// 	ret := fmt.Printf("[-] %v\n", s.Name)
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
	if startupcheck.DoAllStartupChecks() {
		log.Fatalf("Startup checks failed. Exiting ...")
	}

	err := storage.DB.Init()
	if err != nil {
		log.Fatalf("Error on startup: %v", err.Error())
	}

	// TODO: Remove
	_, _, err = netcenter.Registerhost("vm", "vmwiz-test.vsos.ethz.ch")
	if err != nil {
		log.Println(err)
	}

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

	err = netcenter.DeleteDNSEntryByHostname("vmwiz-test.vsos.ethz.ch")
	if err != nil {
		log.Println(err)
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
