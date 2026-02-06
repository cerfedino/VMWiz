package main

import (
	"context"
	"log"
	"os"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/server"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/startupcheck"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/urfave/cli/v3"
)

func main() {
	err := config.AppConfig.Init()
	if err != nil {
		log.Printf("Failed to parse config: %v", err.Error())
		return
	}

	notifier.InitSMTP()

	if startupcheck.DoAllStartupChecks() {
		log.Println("Startup checks failed")
		return
	} else {
		log.Println("Startup checks passed.")
	}

	err = storage.DB.Init()
	if err != nil {
		log.Printf("Error on startup: %v", err.Error())
		return
	}

	auth.Init()
	confirmation.Init()

	cmd := &cli.Command{
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "server",
				Aliases:     []string{},
				Description: "Starts the VMWiz backend server",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					server.StartServer()
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
