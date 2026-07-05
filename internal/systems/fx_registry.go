package systems

import "reflect"

// FXRule defines automated audio and camera feedback for a gameplay event.
type FXRule struct {
	Sound       string  // Audio asset name (e.g. "swing.wav")
	Volume      float32 // Playback volume (0.0 - 1.0)
	ShakeFrames int     // Camera shake duration in frames
}

var fxRegistry = make(map[reflect.Type]FXRule)

func init() {
	RegisterFXRule(HitEvent{}, FXRule{Sound: "hit.wav", Volume: 0.4, ShakeFrames: 4})
	RegisterFXRule(ExplosionEvent{}, FXRule{Sound: "swing.wav", Volume: 0.6, ShakeFrames: 8})
	RegisterFXRule(PickupEvent{}, FXRule{Sound: "coin.wav", Volume: 0.5, ShakeFrames: 0})
}

// RegisterFXRule binds a reflect.Type of an Event to an FXRule.
func RegisterFXRule(evt Event, rule FXRule) {
	if evt == nil {
		return
	}
	t := reflect.TypeOf(evt)
	fxRegistry[t] = rule
}

// GetFXRule retrieves the FXRule associated with an event.
func GetFXRule(evt Event) (FXRule, bool) {
	if evt == nil {
		return FXRule{}, false
	}
	rule, ok := fxRegistry[reflect.TypeOf(evt)]
	return rule, ok
}
