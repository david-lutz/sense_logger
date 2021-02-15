package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/david-lutz/sense_logger/config"
	"github.com/david-lutz/sense_logger/credentials"
	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	var opts struct {
		ConfigFile string `short:"c" long:"config" description:"Config file path" default:"~/.sense_logger.toml"`
		Email      string `long:"email" description:"Sense Account E-mail Address" env:"SENSE_EMAIL"`
		Password   string `long:"password" description:"Sense Account Password" env:"SENSE_PASSWORD"`
	}

	_, err := flags.Parse(&opts)
	fatalOnErr(err)

	cfg, err := config.LoadConfig(opts.ConfigFile, false)
	fatalOnErr(err)

	if opts.Email == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Sense E-mail: ")
		email, err := reader.ReadString('\n')
		fatalOnErr(err)
		opts.Email = strings.TrimSpace(email)
	}

	if opts.Password == "" {
		fmt.Print("Enter Sense Password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		fatalOnErr(err)
		fmt.Println() // ReadPassword doesn't echo the final \n of the password, fake it here
		opts.Password = strings.TrimSpace(string(bytePassword))
	}

	creds, err := credentials.FetchCredentials(opts.Email, opts.Password)
	fatalOnErr(err)

	credFile, err := homedir.Expand(cfg.Sense.CredentialFile)
	fatalOnErr(err)
	err = credentials.WriteCreds(creds, credFile)
	fatalOnErr(err)

	fmt.Println("Successfully retrieved Sense API Credentials")
	fmt.Println("Credentials stored in:", credFile)
}

func fatalOnErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
