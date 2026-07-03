// Command game-test is the desktop entrypoint. It configures the OS window,
// constructs the Ebiten-backed service implementations, injects them into
// [game.App] (the ebiten.Game impl), and hands control to ebiten.RunGame.
//
// Logical resolution is fixed in [game.App.Layout] (320×240); the window is
// 2× for visibility.
//
// Keybinds
//
//   - On startup we call input.LoadOrInit on <UserConfigDir>/game-test/
//     keybinds.json. The file is created with shipped defaults on first
//     run, merged over defaults if present, or ignored (with a warning)
//     if malformed. Either way the game launches.
//
//   - UserConfigDir resolves to ~/Library/Application Support on macOS,
//     ~/.config on Linux, and %AppData% on Windows. If that resolution
//     fails we fall back to shipped defaults without writing anything.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jaredwarren/game-test/assets"
	"github.com/jaredwarren/game-test/internal/balance"
	"github.com/jaredwarren/game-test/internal/game"
	"github.com/jaredwarren/game-test/internal/input"
	ebitenplat "github.com/jaredwarren/game-test/internal/platform/ebiten"
	"github.com/jaredwarren/game-test/internal/services"
)

// configSubdir is the directory under os.UserConfigDir() where per-user
// state for this game lives. Keep it short and filesystem-safe.
const configSubdir = "game-test"

func main() {
	editMap := flag.String("edit", "", "open the in-engine .tmj editor for this map id (e.g. field1); reads/writes assets/maps/<id>.tmj on disk")
	balancePath := flag.String("balance", "", "path to optional balance.json overlay file for dev playtesting")
	flag.Parse()

	if *balancePath != "" {
		data, err := os.ReadFile(*balancePath)
		if err != nil {
			log.Printf("[balance] cannot read balance file %q: %v", *balancePath, err)
		} else {
			if _, err := balance.Load(data); err != nil {
				log.Printf("[balance] cannot parse balance file %q: %v", *balancePath, err)
			} else {
				log.Printf("[balance] loaded balance overlay from %q", *balancePath)
			}
		}
	}

	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("game-test (Ebiten)")
	// Pin Update to the authoritative sim tick rate. Draw still runs at
	// the monitor's FPS; see internal/services/tick.go for the contract.
	ebiten.SetTPS(services.TickRate)

	bindings := loadBindings()

	inp, inpWarnings := ebitenplat.NewInput(bindings)
	for _, w := range inpWarnings {
		log.Printf("[input] %s", w)
	}

	svc := game.Services{
		Input:     inp,
		Audio:     ebitenplat.NewAudio(assets.SoundFS, "sounds/"),
		Assets:    ebitenplat.NewAssetCache(assets.MapFS, "maps/"),
		Clipboard: ebitenplat.NewClipboard(),
	}
	app, err := game.NewApp(svc, *editMap)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

// loadBindings resolves the OS config directory, loads keybinds.json
// (creating it from defaults on first run), logs any non-fatal warnings,
// and returns the effective bindings.
//
// On any failure to resolve the config directory, we fall back to
// in-memory defaults and log — the game always launches.
func loadBindings() *input.Bindings {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("[input] cannot resolve user config dir (%v); using built-in defaults", err)
		return input.DefaultBindings()
	}
	path := filepath.Join(cfgDir, configSubdir, input.DefaultFileName)
	b, warnings, err := input.LoadOrInit(path)
	if err != nil {
		log.Printf("[input] %v; using built-in defaults", err)
		return input.DefaultBindings()
	}
	for _, w := range warnings {
		log.Printf("[input] %s", w)
	}
	return b
}
