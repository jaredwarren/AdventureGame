// Command mapeditor serves a local web-based editor for the game's .tmj maps
// and vector tile art (assets/tiles/*.tile.json).
//
// It is a second editor, not a replacement: the in-engine editor
// (`go run ./cmd/game -edit F-5`) still works and edits the same map files.
//
//	go run ./cmd/mapeditor            # serve, print a tokenized URL
//	go run ./cmd/mapeditor -open      # ... and open the map editor
//	go run ./cmd/mapeditor -open-tiles # ... and open the tile art editor
//	go run ./cmd/mapeditor -check     # validate every map, exit 1 on errors
//
// Everything the browser draws — tile art, marker hit boxes, default properties,
// validation rules — is derived from internal/world at startup, so the editor
// cannot drift from the game. See internal/editorweb.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"syscall"

	"github.com/jaredwarren/game-test/internal/editorweb"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("mapeditor: ")

	var (
		mapsDir     = flag.String("maps", "", "maps directory (default: assets/maps under the module root, or $GAME_MAPS_DIR)")
		addr        = flag.String("addr", "127.0.0.1:7777", "listen address")
		open        = flag.Bool("open", false, "open the map editor in the default browser")
		openTiles   = flag.Bool("open-tiles", false, "open the tile art editor in the default browser")
		check       = flag.Bool("check", false, "validate every map, print a report, and exit non-zero on errors")
		strict      = flag.Bool("strict", false, "treat missing door targets as errors")
		token       = flag.String("token", "", "override the per-launch access token (for scripted testing)")
		allowRemote = flag.Bool("allow-remote", false, "accept non-loopback clients (required when binding a public address)")
		tileStride  = flag.Int("tile-stride", 0, "ticks between recorded animation frames (0 = default)")
		tileFrames  = flag.Int("tile-frames", 0, "animation frames to record per animated tile (0 = default)")
	)
	flag.Parse()

	dir, err := resolveMapsDir(*mapsDir)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := editorweb.New(editorweb.Options{
		MapsDir:     dir,
		Addr:        *addr,
		Token:       *token,
		AllowRemote: *allowRemote,
		Strict:      *strict,
		Anim:        editorweb.AnimOptions(*tileStride, *tileFrames),
		Logger:      log.Default(),
	})
	if err != nil {
		log.Fatal(err)
	}

	if *check {
		os.Exit(runCheck(srv))
	}

	if *openTiles {
		go openBrowser(srv.TilesURL())
	} else if *open {
		go openBrowser(srv.URL())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}

// runCheck validates the whole corpus and returns a process exit code.
func runCheck(srv *editorweb.Server) int {
	report, err := srv.ValidateAll()
	if err != nil {
		log.Print(err)
		return 1
	}

	ids := make([]string, 0, len(report.Maps))
	for id := range report.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		fmt.Printf("%s\n", id)
		for _, issue := range report.Maps[id] {
			fmt.Printf("  %-5s %-24s %s\n", issue.Severity, issue.Code, issue.Message)
		}
	}

	fmt.Printf("\n%d maps checked: %d error(s), %d warning(s)\n",
		report.MapCount, report.ErrorCount, report.WarnCount)
	if report.ErrorCount > 0 {
		return 1
	}
	return 0
}

// openBrowser is best effort: failing to launch a browser must not stop the
// server, since the URL is printed anyway.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	if err := exec.Command(cmd, append(args, url)...).Start(); err != nil {
		log.Printf("could not open a browser (%v); visit the URL above", err)
	}
}
