package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/router"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/rs/cors"
)

func main() {
	storage.DB.Init("")

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
