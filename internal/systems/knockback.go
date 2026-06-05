package systems

import (
	"math"

	"github.com/jaredwarren/game-test/internal/world"
)

const (
	// enemyMeleeKnockbackPx is how far an enemy slides away from the player
	// when struck by sword or torch (world pixels, one axis may absorb more).
	enemyMeleeKnockbackPx = 20.0
	// playerContactKnockbackPx is how far the player slides away from an
	// enemy on a successful contact hit.
	playerContactKnockbackPx = 6.0
)

// knockbackSlide moves (x,y) with hitbox (ww,hh) away from (fromX, fromY).
// moverIsPlayer selects slide rules: player knockback avoids solids and enemies;
// enemy knockback avoids solids and the player.
func knockbackSlide(w *world.World, x, y, ww, hh, fromX, fromY float64, strength float64, moverIsPlayer bool) (float64, float64) {
	tcx := x + ww*0.5
	tcy := y + hh*0.5
	tx := tcx - fromX
	ty := tcy - fromY
	d := math.Hypot(tx, ty)
	if d < 1e-6 {
		return x, y
	}
	dx := tx / d * strength
	dy := ty / d * strength
	if moverIsPlayer {
		return w.SlidePlayerAABB(x, y, ww, hh, dx, dy)
	}
	return w.SlideEnemyAABB(x, y, ww, hh, dx, dy)
}
