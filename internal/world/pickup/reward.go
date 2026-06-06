package pickup

import "github.com/jaredwarren/game-test/internal/progression"

// RewardTarget receives gameplay effects when a pickup is collected.
type RewardTarget interface {
	PickupAddCurrency(amount int)
	PickupHeal(heal int)
	PickupPlayerHP() int
	PickupMaxHP() int
	PickupAddBomb()
	PickupAddSmallKey()
	PickupGrantTorch()
}

func applyCoin(rt RewardTarget) {
	rt.PickupAddCurrency(progression.DefaultEconomy().CoinPickupValue)
}

func applyHeart(rt RewardTarget) {
	heal := progression.DefaultEconomy().HeartPickupHeal
	if heal <= 0 || rt.PickupPlayerHP() >= rt.PickupMaxHP() {
		return
	}
	rt.PickupHeal(heal)
}

func applyBomb(rt RewardTarget) {
	rt.PickupAddBomb()
}

func applySmallKey(rt RewardTarget) {
	rt.PickupAddSmallKey()
}

func applyTorch(rt RewardTarget) {
	rt.PickupGrantTorch()
}
