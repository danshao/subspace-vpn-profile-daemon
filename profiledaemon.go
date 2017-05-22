package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/takama/daemon"

	"gitlab.ecoworkinc.com/Subspace/softetherlib/softether"
	"gitlab.ecoworkinc.com/Subspace/subspace-utility/subspace/repository"
)

const (
	SOFTETHER_ENDPOINT = "localhost"
	SOFTETHER_PASSWORD = "subspace"
	HUB                = "subspace"

	MYSQL_ENDPOINT = "localhost"
	MYSQL_ACCOUNT  = "subspace"
	MYSQL_PASSWORD = "subspace"
	MYSQL_DATABASE = "subspace"
)

const (
	// Name of the service
	name        = "vpnprofile"
	description = "Snapshot VPN user historical data"
)

//	dependencies that are NOT required by the service, but might be used
var dependencies = []string{
	"vpnserver.service", // This daemon depends on soft-ether vpn server
}

var stdlog, errlog *log.Logger

// Service has embedded daemon
type Service struct {
	daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	usage := "Usage: myservice install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	// Do something, call your goroutines, etc
	vpnServer := softether.SoftEther{
		IP:       SOFTETHER_ENDPOINT,
		Password: SOFTETHER_PASSWORD,
		Hub:      HUB,
	}
	profileSnapshotRepo := repository.MysqlProfileSnapshotRepository{
		Host:         MYSQL_ENDPOINT,
		Account:      MYSQL_ACCOUNT,
		Password:     MYSQL_PASSWORD,
		DatabaseName: MYSQL_DATABASE,
	}
	profileRepo := repository.MysqlProfileRepository{
		Host:         MYSQL_ENDPOINT,
		Account:      MYSQL_ACCOUNT,
		Password:     MYSQL_PASSWORD,
		DatabaseName: MYSQL_DATABASE,
	}

	runner := ProfileDaemonRunner{
		Server:                    vpnServer,
		ProfileSnapshotRepository: profileSnapshotRepo,
		ProfileRepository:         profileRepo,
	}
	runner.Start()

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case killSignal := <-interrupt:
			stdlog.Println("Got signal:", killSignal)
			runner.Stop()
			if killSignal == os.Interrupt {
				return "Daemon was interruped by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}

	// never happen, but need to complete code
	return usage, nil
}

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
	srv, err := daemon.New(name, description, dependencies...)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}
