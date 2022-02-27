package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/purwandi/consulat/consulat"
)

var (
	errChan = make(chan error, 10)
)

func main() {
	// print basic info
	log.Println("consulat mode ", os.Getenv("CONSULAT_MODE"))
	log.Println("consulat auth method ", os.Getenv("CONSUL_AUTH_METHOD"))

	log.Println("initialize consulat")
	consulat, err := consulat.New(
		os.Getenv("CONSUL_AUTH_METHOD"),
		os.Getenv("KUBERNETES_JWT_SECRET"),
		os.Getenv("CONSUL_TOKEN_FILE"))
	if err != nil {
		panic(err)
	}

	if err := consulat.Login(); err != nil {
		panic(err)
	}

	// stop process if mode is not sidecar
	if os.Getenv("CONSULAT_MODE") != "sidecar" {
		log.Println("token successfull retrieve")
		os.Exit(0)
	}

	if consulat.ACLToken.ExpirationTime == nil {
		log.Println("expired is not set, stoping")
		os.Exit(0)
	}

	go func() {
		errChan <- consulat.Renew()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			if err != nil {
				log.Fatal(err)
			}
		case s := <-signalChan:
			log.Println(fmt.Sprintf("captured %v. exiting...", s))

			log.Println("logging out consul and removing existing consul token")
			consulat.Logout()

			log.Println("successfull terminated consulat")
			os.Exit(0)
		}
	}
}
