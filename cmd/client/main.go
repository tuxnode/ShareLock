package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cs161-staff/project2-starter-code/internal/client/app"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Parse global flags (before subcommand)
	kvAddr := os.Getenv("SHARELOCK_KV_ADDR")
	kvTLS := true
	if v := os.Getenv("SHARELOCK_KV_TLS"); v == "false" || v == "0" {
		kvTLS = false
	}

	// Connect to KV server if configured
	if kvAddr != "" {
		app.Connect(kvAddr, kvTLS)
		defer app.Disconnect()
	}

	cli := &app.Client{}

	switch os.Args[1] {
	case "inituser":
		cmd := flag.NewFlagSet("inituser", flag.ExitOnError)
		username := cmd.String("username", "", "")
		password := cmd.String("password", "", "")
		cmd.Parse(os.Args[2:])
		if err := cli.InitUser(*username, *password); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "getuser":
		cmd := flag.NewFlagSet("getuser", flag.ExitOnError)
		username := cmd.String("username", "", "")
		password := cmd.String("password", "", "")
		cmd.Parse(os.Args[2:])
		if err := cli.GetUser(*username, *password); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "storefile":
		cmd := flag.NewFlagSet("storefile", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		content := cmd.String("content", "", "")
		cmd.Parse(os.Args[2:])
		if err := cli.StoreFile(*filename, []byte(*content)); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "loadfile":
		cmd := flag.NewFlagSet("loadfile", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		cmd.Parse(os.Args[2:])
		data, err := cli.LoadFile(*filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(data))

	case "appendtofile":
		cmd := flag.NewFlagSet("appendtofile", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		content := cmd.String("content", "", "")
		cmd.Parse(os.Args[2:])
		if err := cli.AppendToFile(*filename, []byte(*content)); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "createinvitation":
		cmd := flag.NewFlagSet("createinvitation", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		recipient := cmd.String("recipient", "", "")
		cmd.Parse(os.Args[2:])
		invite, err := cli.CreateInvitation(*filename, *recipient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(invite.String())

	case "acceptinvitation":
		cmd := flag.NewFlagSet("acceptinvitation", flag.ExitOnError)
		sender := cmd.String("sender", "", "")
		invitation := cmd.String("invitation", "", "")
		filename := cmd.String("filename", "", "")
		cmd.Parse(os.Args[2:])
		invUUID, err := uuid.Parse(*invitation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid invitation UUID: %v\n", err)
			os.Exit(1)
		}
		if err := cli.AcceptInvitation(*sender, invUUID, *filename); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "revokeaccess":
		cmd := flag.NewFlagSet("revokeaccess", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		recipient := cmd.String("recipient", "", "")
		cmd.Parse(os.Args[2:])
		if err := cli.RevokeAccess(*filename, *recipient); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "read":
		cmd := flag.NewFlagSet("read", flag.ExitOnError)
		filename := cmd.String("filename", "", "")
		address := cmd.String("address", "localhost:8080", "")
		tlsEnabled := cmd.Bool("tls", true, "use TLS encryption")
		cmd.Parse(os.Args[2:])
		if err := cli.ReadFileTLS(*filename, *address, *tlsEnabled); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: %s <command> [flags]

Environment:
  SHARELOCK_KV_ADDR   KV server address (e.g. localhost:8080)
  SHARELOCK_KV_TLS    enable TLS for KV server (default true)

Commands:
  inituser -username <name> -password <pw>
  getuser -username <name> -password <pw>
  storefile -filename <name> -content <data>
  loadfile -filename <name>
  appendtofile -filename <name> -content <data>
  createinvitation -filename <name> -recipient <user>
  acceptinvitation -sender <user> -invitation <uuid> -filename <name>
  revokeaccess -filename <name> -recipient <user>
  read -filename <name> [-address <host:port>] [-tls=true]
`, os.Args[0])
}
