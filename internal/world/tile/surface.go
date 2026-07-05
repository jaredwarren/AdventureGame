package tile

type SurfaceType string

const (
	SurfaceNormal SurfaceType = "normal"
	SurfaceIce    SurfaceType = "ice"
	SurfaceMud    SurfaceType = "mud"
	SurfaceLava   SurfaceType = "lava"
	SurfaceWater  SurfaceType = "water"
)

// SurfaceDef describes physical surface properties of a tile type.
type SurfaceDef struct {
	Type            SurfaceType
	SpeedMultiplier float64
	Friction        float64
	HazardDamage    int
	HazardInterval  int
	Tags            []string
}

var (
	DefaultSurface = SurfaceDef{
		Type:            SurfaceNormal,
		SpeedMultiplier: 1.0,
		Friction:        1.0,
		HazardDamage:    0,
		HazardInterval:  0,
	}

	MudSurface = SurfaceDef{
		Type:            SurfaceMud,
		SpeedMultiplier: 0.5,
		Friction:        0.8,
		Tags:            []string{"slow"},
	}

	IceSurface = SurfaceDef{
		Type:            SurfaceIce,
		SpeedMultiplier: 1.1,
		Friction:        0.1,
		Tags:            []string{"slippery"},
	}

	LavaSurface = SurfaceDef{
		Type:            SurfaceLava,
		SpeedMultiplier: 0.7,
		Friction:        1.0,
		HazardDamage:    1,
		HazardInterval:  30,
		Tags:            []string{"hazard", "fire"},
	}
)

// SurfaceForGID returns the SurfaceDef for a given tile GID.
func SurfaceForGID(gid int) SurfaceDef {
	def := DefOf(gid)
	if def.Water {
		return MudSurface
	}
	switch def.Name {
	case "mud":
		return MudSurface
	case "ice":
		return IceSurface
	case "lava":
		return LavaSurface
	default:
		return DefaultSurface
	}
}
