package progression

// Economy holds shop prices and reward amounts for coins, pickups, and shrines.
type Economy struct {
	ShopVitalityCost  int
	ShopResolveCost   int
	ShopMightCost     int
	ShopFullHealCost  int
	ShopBombCost      int
	ShopTorchCost     int
	BossKillCoinBonus int
	CoinPickupValue   int
	HeartPickupHeal   int
	ShrineHealAmount  int
	ShopShieldCost        int
	ShopShieldUpgradeCost int
}

var defaultEconomy = Economy{
	ShopVitalityCost:  10,
	ShopResolveCost:   8,
	ShopMightCost:     12,
	ShopFullHealCost:  4,
	ShopBombCost:      3,
	ShopTorchCost:     5,
	BossKillCoinBonus: 25,
	CoinPickupValue:   1,
	HeartPickupHeal:   1,
	ShrineHealAmount:  2,
	ShopShieldCost:        8,
	ShopShieldUpgradeCost: 15,
}

// DefaultEconomy returns the canonical economy tuning defaults.
func DefaultEconomy() Economy {
	return defaultEconomy
}
