package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cs161-staff/project2-starter-code/internal/client/app"
	"github.com/cs161-staff/project2-starter-code/internal/client/config"
	"github.com/google/uuid"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Handle --help/-h globally
	if os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	// Handle version
	if os.Args[1] == "version" || os.Args[1] == "--version" {
		fmt.Printf("sharelock %s\n", version)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: config load failed: %v\n", err)
		os.Exit(1)
	}

	// Handle host management commands
	if os.Args[1] == "host" || os.Args[1] == "hosts" {
		hostCmd(cfg, os.Args[2:])
		return
	}

	// Connect to KV server via config (except for 'read')
	if os.Args[1] != "read" {
		hostName := "default"
		if h := os.Getenv("SHARELOCK_HOST"); h != "" {
			hostName = h
		}
		entry, ok := cfg.Get(hostName)
		if ok && entry.Addr != "" {
			app.Connect(entry.Addr, entry.TLS)
			defer app.Disconnect()
		}
	}

	cli := &app.Client{}

	switch os.Args[1] {
	case "init":
		cmdInit(cli, os.Args[2:])
	case "login":
		cmdLogin(cli, os.Args[2:])
	case "store":
		cmdStore(cli, os.Args[2:])
	case "load":
		cmdLoad(cli, os.Args[2:])
	case "append":
		cmdAppend(cli, os.Args[2:])
	case "share":
		cmdShare(cli, os.Args[2:])
	case "accept":
		cmdAccept(cli, os.Args[2:])
	case "revoke":
		cmdRevoke(cli, os.Args[2:])
	case "list":
		cmdList(cli, os.Args[2:])
	case "read":
		cmdRead(cli, os.Args[2:])

	// Legacy command aliases
	case "inituser", "getuser", "storefile", "loadfile", "appendtofile",
		"createinvitation", "acceptinvitation", "revokeaccess":
		runLegacy(os.Args[1], cli, os.Args[2:])

	case "help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "Run 'sharelock help' to see available commands.\n")
		os.Exit(1)
	}
}

func cmdInit(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	username := fs.String("u", "", "username")
	password := fs.String("p", "", "password")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock init -u <username> -p <password>")
		fmt.Fprintln(os.Stderr, "\nCreate a new user account.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Error: username and password are required")
		fs.Usage()
		os.Exit(1)
	}

	if err := cli.InitUser(*username, *password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdLogin(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	username := fs.String("u", "", "username")
	password := fs.String("p", "", "password")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock login -u <username> -p <password>")
		fmt.Fprintln(os.Stderr, "\nLogin as an existing user.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Error: username and password are required")
		fs.Usage()
		os.Exit(1)
	}

	if err := cli.GetUser(*username, *password); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdStore(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("store", flag.ExitOnError)
	filepathFlag := fs.String("f", "", "local file path")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock store -f <filepath>")
		fmt.Fprintln(os.Stderr, "\nStore a file by reading its content from disk (overwrites if exists).")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filepathFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: file path is required")
		fs.Usage()
		os.Exit(1)
	}

	content, err := os.ReadFile(*filepathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read file '%s': %v\n", *filepathFlag, err)
		os.Exit(1)
	}

	filename := filepath.Base(*filepathFlag)

	if err := cli.StoreFile(filename, content); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdLoad(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	filename := fs.String("f", "", "filename")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock load -f <filename>")
		fmt.Fprintln(os.Stderr, "\nLoad and print file contents.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filename == "" {
		fmt.Fprintln(os.Stderr, "Error: filename is required")
		fs.Usage()
		os.Exit(1)
	}

	data, err := cli.LoadFile(*filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

func cmdAppend(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("append", flag.ExitOnError)
	filename := fs.String("f", "", "filename")
	content := fs.String("c", "", "content")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock append -f <filename> -c <content>")
		fmt.Fprintln(os.Stderr, "\nAppend content to an existing file.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filename == "" {
		fmt.Fprintln(os.Stderr, "Error: filename is required")
		fs.Usage()
		os.Exit(1)
	}

	if err := cli.AppendToFile(*filename, []byte(*content)); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdShare(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("share", flag.ExitOnError)
	filename := fs.String("f", "", "filename")
	recipient := fs.String("r", "", "recipient username")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock share -f <filename> -r <username>")
		fmt.Fprintln(os.Stderr, "\nShare a file with another user. Prints invitation UUID.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filename == "" || *recipient == "" {
		fmt.Fprintln(os.Stderr, "Error: filename and recipient are required")
		fs.Usage()
		os.Exit(1)
	}

	invite, err := cli.CreateInvitation(*filename, *recipient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(invite.String())
}

func cmdAccept(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("accept", flag.ExitOnError)
	sender := fs.String("s", "", "sender username")
	invitation := fs.String("i", "", "invitation UUID")
	filename := fs.String("f", "", "filename to save as")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock accept -s <sender> -i <uuid> -f <filename>")
		fmt.Fprintln(os.Stderr, "\nAccept a file sharing invitation.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *sender == "" || *invitation == "" || *filename == "" {
		fmt.Fprintln(os.Stderr, "Error: sender, invitation UUID, and filename are required")
		fs.Usage()
		os.Exit(1)
	}

	invUUID, err := uuid.Parse(*invitation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid invitation UUID: %v\n", err)
		os.Exit(1)
	}

	if err := cli.AcceptInvitation(*sender, invUUID, *filename); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdRevoke(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("revoke", flag.ExitOnError)
	filename := fs.String("f", "", "filename")
	recipient := fs.String("r", "", "recipient username")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock revoke -f <filename> -r <username>")
		fmt.Fprintln(os.Stderr, "\nRevoke a user's access to a file.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filename == "" || *recipient == "" {
		fmt.Fprintln(os.Stderr, "Error: filename and recipient are required")
		fs.Usage()
		os.Exit(1)
	}

	if err := cli.RevokeAccess(*filename, *recipient); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdList(cli *app.Client, args []string) {
	files := cli.ListFiles()
	if len(files) == 0 {
		fmt.Println("No files stored.")
		return
	}
	for _, f := range files {
		fmt.Println(f)
	}
}

func cmdRead(cli *app.Client, args []string) {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	filename := fs.String("f", "", "filename")
	address := fs.String("a", "localhost:8080", "server address")
	tlsEnabled := fs.Bool("tls", true, "use TLS encryption")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sharelock read -f <filename> [-a <host:port>] [-tls=true]")
		fmt.Fprintln(os.Stderr, "\nRead a file via TLS-encrypted streaming.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if *filename == "" {
		fmt.Fprintln(os.Stderr, "Error: filename is required")
		fs.Usage()
		os.Exit(1)
	}

	if err := cli.ReadFileTLS(*filename, *address, *tlsEnabled); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

// runLegacy handles old command names for backward compatibility
func runLegacy(cmd string, cli *app.Client, args []string) {
	fmt.Fprintf(os.Stderr, "Warning: '%s' is deprecated. Use 'sharelock %s' instead.\n\n", cmd, legacyAlias(cmd))

	switch cmd {
	case "inituser":
		cmdInit(cli, args)
	case "getuser":
		cmdLogin(cli, args)
	case "storefile":
		cmdStore(cli, args)
	case "loadfile":
		cmdLoad(cli, args)
	case "appendtofile":
		cmdAppend(cli, args)
	case "createinvitation":
		cmdShare(cli, args)
	case "acceptinvitation":
		cmdAccept(cli, args)
	case "revokeaccess":
		cmdRevoke(cli, args)
	}
}

func legacyAlias(cmd string) string {
	switch cmd {
	case "inituser":
		return "init"
	case "getuser":
		return "login"
	case "storefile":
		return "store"
	case "loadfile":
		return "load"
	case "appendtofile":
		return "append"
	case "createinvitation":
		return "share"
	case "acceptinvitation":
		return "accept"
	case "revokeaccess":
		return "revoke"
	default:
		return cmd
	}
}

func hostCmd(cfg *config.Config, args []string) {
	if len(args) == 0 {
		hostUsage()
		return
	}

	if args[0] == "--help" || args[0] == "-h" {
		hostUsage()
		return
	}

	switch args[0] {
	case "list", "ls":
		hosts := cfg.List()
		names := make([]string, 0, len(hosts))
		for n := range hosts {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			h := hosts[n]
			tlsStr := "tls"
			if !h.TLS {
				tlsStr = "plain"
			}
			mark := " "
			if n == "default" {
				mark = "*"
			}
			fmt.Printf("%s %-12s %s (%s)\n", mark, n, h.Addr, tlsStr)
		}

	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: sharelock host add <name> <addr> [--tls=false]")
			os.Exit(1)
		}
		name := args[1]
		addr := args[2]
		tls := true
		for i := 3; i < len(args); i++ {
			if args[i] == "--tls=false" || args[i] == "--no-tls" {
				tls = false
			}
		}
		cfg.Set(name, addr, tls)
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: save failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "rm", "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: sharelock host rm <name>")
			os.Exit(1)
		}
		cfg.Remove(args[1])
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: save failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	case "default", "use":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: sharelock host use <name>")
			os.Exit(1)
		}
		entry, ok := cfg.Get(args[1])
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: host '%s' not found\n", args[1])
			os.Exit(1)
		}
		cfg.Set("default", entry.Addr, entry.TLS)
		cfg.Remove(args[1])
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: save failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ok")

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown host command '%s'\n\n", args[0])
		hostUsage()
	}
}

func hostUsage() {
	fmt.Fprintln(os.Stderr, `Usage: sharelock host <command> [args]

Commands:
  list, ls                  List all configured hosts
  add <name> <addr>         Add a host (use --tls=false for plain TCP)
  rm <name>                 Remove a host
  use <name>                Set a host as default

Examples:
  sharelock host add dev localhost:8080 --tls=false
  sharelock host use dev
  sharelock host list`)
}

func printUsage() {
	cmd := os.Args[0]
	align := func(name, desc string) string {
		return fmt.Sprintf("  %-16s %s", name, desc)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`ShareLock - End-to-end encrypted file sharing

Usage:
  %s <command> [flags]

Commands:
`, cmd))

	sb.WriteString(align("init", "Create a new user account") + "\n")
	sb.WriteString(align("login", "Login as existing user") + "\n")
	sb.WriteString(align("store", "Store a file from disk (overwrites if exists)") + "\n")
	sb.WriteString(align("load", "Load and print file contents") + "\n")
	sb.WriteString(align("append", "Append content to a file") + "\n")
	sb.WriteString(align("share", "Share a file with another user") + "\n")
	sb.WriteString(align("accept", "Accept a file sharing invitation") + "\n")
	sb.WriteString(align("revoke", "Revoke a user's access") + "\n")
	sb.WriteString(align("list", "List all stored files") + "\n")
	sb.WriteString(align("read", "Read file via TLS streaming") + "\n")
	sb.WriteString(align("host", "Manage server connections") + "\n")
	sb.WriteString(align("version", "Show version") + "\n")
	sb.WriteString(align("help", "Show this help") + "\n")

	sb.WriteString(`
Flags:
  -u    Username
  -p    Password
  -f    Filename
  -c    Content
  -r    Recipient
  -s    Sender
  -i    Invitation UUID
  -a    Server address

Quick Examples:
  # Create account
  sharelock init -u alice -p secret

  # Store a file from disk
  sharelock store -f ./hello.txt

  # Load a file
  sharelock load -f hello.txt

  # Share with someone
  sharelock share -f hello.txt -r bob

  # Accept invitation
  sharelock accept -s alice -i <uuid> -f hello.txt

  # Revoke access
  sharelock revoke -f hello.txt -r bob

Environment:
  SHARELOCK_HOST    Select host from config (default: "default")

Config:
  ~/.config/sharelock/.hosts  (fallback: ~/.hosts)
`)

	fmt.Fprint(os.Stderr, sb.String())
}
