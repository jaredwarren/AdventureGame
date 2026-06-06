package progression

import "testing"

func TestDefaultEconomy_Values(t *testing.T) {
	t.Parallel()
	e := DefaultEconomy()
	if e.ShopVitalityCost != 10 || e.ShopResolveCost != 8 || e.ShopMightCost != 12 {
		t.Errorf("shop upgrade costs: %+v", e)
	}
	if e.ShopFullHealCost != 4 || e.ShopBombCost != 3 || e.ShopTorchCost != 5 {
		t.Errorf("shop item costs: %+v", e)
	}
	if e.BossKillCoinBonus != 25 {
		t.Errorf("BossKillCoinBonus = %d", e.BossKillCoinBonus)
	}
	if e.CoinPickupValue != 1 || e.HeartPickupHeal != 1 || e.ShrineHealAmount != 2 {
		t.Errorf("rewards: coin=%d heart=%d shrine=%d", e.CoinPickupValue, e.HeartPickupHeal, e.ShrineHealAmount)
	}
}
