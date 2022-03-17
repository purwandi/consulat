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

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	var jwtKey = GetEnv("KUBERNETES_JWT_SECRET", "/run/secrets/kubernetes.io/serviceaccount/token")
	var consulMode = GetEnv("CONSULAT_MODE", "sidecar")
	var consulAuthMethod = GetEnv("CONSULAT_AUTH_METHOD", "kubernetes-consul-jwt")
	var consulTokenFile = GetEnv("CONSULAT_TOKEN_FILE", "/run/secrets/consul/token")

	// print basic info
	log.Println("consulat mode ", consulMode)
	log.Println("consulat auth method ", consulAuthMethod)
	log.Println("initialize consulat")

	consulat, err := consulat.New(consulAuthMethod, jwtKey, consulTokenFile)
	if err != nil {
		panic(err)
	}

	if err := consulat.Login(); err != nil {
		panic(err)
	}

	// stop process if mode is not sidecar
	if consulMode != "sidecar" {
		log.Println("consulat mode is init container, shutting down ...")
		os.Exit(0)
	}

	if consulat.ACLToken.ExpirationTime == nil {
		log.Println("token don't have expired, shutting down ...")
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

			log.Println("shutting down ...")
			os.Exit(0)
		}
	}
}
