package ebitenplat

import (
	"bytes"
	"embed"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

	"github.com/jaredwarren/game-test/internal/services"
)

const audioSampleRate = 48000

// Audio implements services.Audio using Ebiten's audio/wav pipeline. SFX are
// decoded on play; decoded bytes are not cached (WAVs are small). If a hot
// path emerges, add a decode-once cache keyed by clipID.
type Audio struct {
	ctx     *audio.Context
	players []*audio.Player
	soundFS embed.FS
	// prefix is prepended to clipID to form the embed path, e.g. "sounds/".
	prefix string
}

// NewAudio constructs an Audio backed by the given embed.FS. prefix is the
// directory inside the FS where clipIDs resolve (use "sounds/" for
// assets.SoundFS). If ebiten hasn't created an audio context elsewhere, this
// constructor owns the single process-wide instance.
func NewAudio(soundFS embed.FS, prefix string) *Audio {
	return &Audio{
		ctx:     audio.NewContext(audioSampleRate),
		soundFS: soundFS,
		prefix:  prefix,
	}
}

// Tick prunes finished players. Call once per frame from the app loop.
func (a *Audio) Tick() {
	alive := a.players[:0]
	for _, p := range a.players {
		if p.IsPlaying() {
			alive = append(alive, p)
			continue
		}
		_ = p.Close()
	}
	a.players = alive
}

func (a *Audio) Play(clipID string, volume float64) {
	if a.ctx == nil {
		return
	}
	b, err := a.soundFS.ReadFile(a.prefix + clipID)
	if err != nil {
		return
	}
	d, err := wav.DecodeF32(bytes.NewReader(b))
	if err != nil {
		return
	}
	p, err := a.ctx.NewPlayerF32(d)
	if err != nil {
		return
	}
	p.SetVolume(volume)
	p.Play()
	a.players = append(a.players, p)
}

var _ services.Audio = (*Audio)(nil)
