package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wasson-ece/logcurse/internal/editor"
	"github.com/wasson-ece/logcurse/internal/viewer/tui"
	"github.com/wasson-ece/logcurse/internal/viewer/web"
)

// Set via -ldflags at build time: -ldflags "-X main.version=v1.0.0"
var version = "dev"

func main() {
	rangeFlag := flag.String("n", "", "Add a comment on a line range (sed-style, e.g. '1,10p' or '42')")
	serveFlag := flag.Bool("serve", false, "Serve a web viewer")
	portFlag := flag.Int("port", 8080, "Port for web server (used with --serve)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: logcurse [flags] <file>\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  logcurse <file>                  View file with comments in TUI\n")
		fmt.Fprintf(os.Stderr, "  logcurse -n '1,10p' <file>       Add a comment on lines 1-10\n")
		fmt.Fprintf(os.Stderr, "  logcurse -n 42 <file>            Add a comment on line 42\n")
		fmt.Fprintf(os.Stderr, "  logcurse --serve <file|dir>       Serve a web viewer\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("logcurse %s\n", version)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	sourceFile := flag.Arg(0)

	// Verify path exists
	info, err := os.Stat(sourceFile)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: %q not found\n", sourceFile)
		os.Exit(1)
	}

	switch {
	case *rangeFlag != "":
		if info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: cannot use -n with a directory\n")
			os.Exit(1)
		}
		if err := editor.AddComment(sourceFile, *rangeFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *serveFlag:
		if info.IsDir() {
			if err := web.ServeDirectory(sourceFile, *portFlag, version); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			if err := runWebServer(sourceFile, *portFlag); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	default:
		if info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: cannot use TUI viewer with a directory; use --serve instead\n")
			os.Exit(1)
		}
		if err := runTUI(sourceFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func runTUI(sourceFile string) error {
	return tui.Run(sourceFile)
}

func runWebServer(sourceFile string, port int) error {
	return web.Serve(sourceFile, port, version)
}
