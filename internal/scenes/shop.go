package scenes

import (
	"fmt"
	"image/color"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/services"
	"github.com/jaredwarren/game-test/internal/world"
)

type ShopItem struct {
	Name        string
	Cost        int
	Description string
	Buy         func(w *world.World) bool
	IsEnabled   func(w *world.World) bool
}

type ShopScene struct {
	selectedIndex int
}

func newShopScene() Scene {
	return &ShopScene{selectedIndex: 0}
}

func (s *ShopScene) ID() SceneID { return SceneShop }

func (s *ShopScene) Enter(ctx GameContext, params map[string]any) error {
	s.selectedIndex = 0
	return nil
}

func (s *ShopScene) Exit(ctx GameContext) error { return nil }

func (s *ShopScene) getItems(w *world.World) []ShopItem {
	cfg := progression.DefaultConfig()
	econ := progression.DefaultEconomy()
	return []ShopItem{
		{
			Name:        "Upgrade Vitality",
			Cost:        econ.ShopVitalityCost,
			Description: fmt.Sprintf("+%d Max HP", cfg.HPPerVitality),
			Buy: func(w *world.World) bool {
				w.Stats.Vitality++
				w.HP += cfg.HPPerVitality
				return true
			},
			IsEnabled: func(w *world.World) bool { return true },
		},
		{
			Name:        "Upgrade Resolve",
			Cost:        econ.ShopResolveCost,
			Description: fmt.Sprintf("+%d Max Stamina", cfg.StaminaPerResolve),
			Buy: func(w *world.World) bool {
				w.Stats.Resolve++
				return true
			},
			IsEnabled: func(w *world.World) bool { return true },
		},
		{
			Name:        "Upgrade Might",
			Cost:        econ.ShopMightCost,
			Description: fmt.Sprintf("+%d Sword Damage", cfg.DamagePerMight),
			Buy: func(w *world.World) bool {
				w.Stats.Might++
				return true
			},
			IsEnabled: func(w *world.World) bool { return true },
		},
		{
			Name:        "Heal Full HP",
			Cost:        econ.ShopFullHealCost,
			Description: "Restore all hearts",
			Buy: func(w *world.World) bool {
				w.HP = w.MaxHP()
				return true
			},
			IsEnabled: func(w *world.World) bool {
				return w.HP < w.MaxHP()
			},
		},
		{
			Name:        "Buy Bomb",
			Cost:        econ.ShopBombCost,
			Description: "+1 Bomb",
			Buy: func(w *world.World) bool {
				if w.Bombs >= w.MaxBombsCarry() {
					return false
				}
				w.Bombs++
				return true
			},
			IsEnabled: func(w *world.World) bool {
				return w.Bombs < w.MaxBombsCarry()
			},
		},
		{
			Name:        "Buy Torch",
			Cost:        econ.ShopTorchCost,
			Description: "See in the dark night",
			Buy: func(w *world.World) bool {
				w.GrantItem("torch")
				return true
			},
			IsEnabled: func(w *world.World) bool {
				return !w.HasItem("torch")
			},
		},
	}
}

func (s *ShopScene) Update(ctx GameContext) error {
	in := ctx.Input()
	sess := ctx.Session()
	w := sess.World
	if w == nil {
		return nil
	}

	items := s.getItems(w)
	numItems := len(items)

	if in.JustPressed(services.ActionMoveUp) {
		s.selectedIndex = (s.selectedIndex - 1 + numItems) % numItems
	}
	if in.JustPressed(services.ActionMoveDown) {
		s.selectedIndex = (s.selectedIndex + 1) % numItems
	}

	if in.JustPressed(services.ActionConfirm) {
		item := items[s.selectedIndex]
		if w.Currency >= item.Cost && item.IsEnabled(w) {
			w.Currency -= item.Cost
			if item.Buy(w) {
				ctx.Audio().Play("pickup.wav", 0.35)
			} else {
				w.Currency += item.Cost // Refund if buy failed
				ctx.Audio().Play("hit.wav", 0.25)
			}
		} else {
			ctx.Audio().Play("hit.wav", 0.25)
		}
	}

	if in.JustPressed(services.ActionCancel) || in.JustPressed(services.ActionPause) || in.JustPressed(services.ActionInteract) {
		ctx.Manager().PopOverlay()
	}

	return nil
}

func (s *ShopScene) Draw(ctx GameContext) {
	sess := ctx.Session()
	r := ctx.Renderer()
	w := sess.World

	DrawOverlayDim(r)

	// Draw shop overlay card
	px, py := float32(60), float32(30)
	pw, ph := float32(200), float32(180)

	// Blue-black translucent backdrop
	r.FillRect(px, py, pw, ph, color.RGBA{10, 15, 30, 235})
	// Golden border
	r.StrokeRect(px, py, pw, ph, 1.5, color.RGBA{255, 215, 0, 200})

	// Title
	r.DrawText(int(px)+50, int(py)+8, "SHRINE UPGRADES")
	r.StrokeLine(px+10, py+22, px+pw-10, py+22, 1, color.RGBA{255, 215, 0, 100})

	// Currency
	coinsText := "Coins: 0"
	if w != nil {
		coinsText = fmt.Sprintf("Coins: %d", w.Currency)
	}
	r.DrawText(int(px)+15, int(py)+28, coinsText)

	if w == nil {
		return
	}

	items := s.getItems(w)
	startY := int(py) + 46
	lineH := 16

	for i, item := range items {
		isActive := (i == s.selectedIndex)
		enabled := item.IsEnabled(w) && w.Currency >= item.Cost

		prefix := "  "
		if isActive {
			prefix = "> "
		}

		itemColor := color.RGBA{200, 200, 200, 255} // standard available
		if isActive {
			itemColor = color.RGBA{255, 215, 0, 255} // highlighted active
		}
		if !enabled {
			if isActive {
				itemColor = color.RGBA{180, 130, 0, 255} // active but unavailable
			} else {
				itemColor = color.RGBA{100, 100, 100, 255} // unavailable greyed out
			}
		}

		displayText := fmt.Sprintf("%s%-18s [%2dc]", prefix, item.Name, item.Cost)
		r.DrawText(int(px)+10, startY+i*lineH, displayText)
		// We manually color by drawing over it or relying on active visual markers
		_ = itemColor // unused in flat DrawText but kept for design logic
	}

	// Bottom separator
	r.StrokeLine(px+10, py+ph-30, px+pw-10, py+ph-30, 1, color.RGBA{255, 215, 0, 100})

	// Description
	descText := items[s.selectedIndex].Description
	r.DrawText(int(px)+15, int(py+ph)-22, descText)

	if sess.ShowDebugOverlay {
		DrawDebugOverlay(r, sess, s.ID(), DebugOverlayExtras{})
	}
}
