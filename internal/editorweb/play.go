package editorweb

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gameModule is the go.mod path this editor belongs to. Play walks up from the
// maps directory until it finds a Makefile next to that module, so `make run`
// executes in the repo even when -maps points at assets/maps.
const gameModule = "github.com/jaredwarren/game-test"

func (s *Server) handlePlay(w http.ResponseWriter, r *http.Request) {
	restarted, pid, dir, err := s.startPlay()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "play_failed", "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"pid":       pid,
		"restarted": restarted,
		"dir":       dir,
		"cmd":       []string{"make", "run"},
	})
}

func (s *Server) startPlay() (restarted bool, pid int, dir string, err error) {
	dir, err = findPlayDir(s.store.Root)
	if err != nil {
		return false, 0, "", err
	}

	s.playMu.Lock()
	defer s.playMu.Unlock()

	if s.playCmd != nil && s.playCmd.Process != nil {
		restarted = true
		s.killPlayLocked()
	}

	cmd := exec.Command("make", "run")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setPlayProcGroup(cmd)
	if err := cmd.Start(); err != nil {
		return false, 0, dir, fmt.Errorf("start make run: %w", err)
	}
	s.playCmd = cmd
	pid = cmd.Process.Pid
	s.logger.Printf("map editor: make run pid=%d in %s", pid, dir)

	go func() {
		waitErr := cmd.Wait()
		s.playMu.Lock()
		if s.playCmd == cmd {
			s.playCmd = nil
		}
		s.playMu.Unlock()
		if waitErr != nil {
			s.logger.Printf("map editor: make run exited: %v", waitErr)
		}
	}()
	return restarted, pid, dir, nil
}

func (s *Server) stopPlay() {
	s.playMu.Lock()
	defer s.playMu.Unlock()
	s.killPlayLocked()
}

func (s *Server) killPlayLocked() {
	if s.playCmd == nil {
		return
	}
	killPlayProcGroup(s.playCmd)
	s.playCmd = nil
}

func findPlayDir(mapsDir string) (string, error) {
	dir, err := filepath.Abs(mapsDir)
	if err != nil {
		return "", err
	}
	for {
		makefile := filepath.Join(dir, "Makefile")
		if st, err := os.Stat(makefile); err == nil && !st.IsDir() && declaresGameModule(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no Makefile for %s above %s", gameModule, mapsDir)
		}
		dir = parent
	}
}

func declaresGameModule(goMod string) bool {
	f, err := os.Open(goMod)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if name, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(name) == gameModule
		}
	}
	return false
}
