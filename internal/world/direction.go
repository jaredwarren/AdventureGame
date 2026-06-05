package world

// Dir represents a cardinal facing direction for players and entities.
type Dir = int

const (
	DirDown  Dir = 0
	DirUp    Dir = 1
	DirLeft  Dir = 2
	DirRight Dir = 3
)

// DirLabel returns a human-readable name for a direction.
func DirLabel(d Dir) string {
	switch d {
	case DirDown:
		return "down"
	case DirUp:
		return "up"
	case DirLeft:
		return "left"
	case DirRight:
		return "right"
	default:
		return "?"
	}
}
