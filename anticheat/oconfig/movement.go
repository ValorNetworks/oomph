package oconfig

type MovementOpts struct {
	// PredictionThreshold raises a flag; it never controls movement correction.
	PredictionThreshold float32 `json:"prediction_threshold" comment:"Distance in blocks between continuous prediction and reported position required to flag Movement A. No movement correction is performed."`
	// LimitAllVelocity is a boolean that represents if the Oomph proxy should limit the prediction velocity to a certain limit. This prevents exploiters from setting a large
	// velocity value when exempted to cause crashes or lag.
	LimitAllVelocity bool `json:"limit_all_velocity" comment:"Should Oomph limit the prediction velocity to a certain limit? This prevents exploiters from setting a large velocity value when exempted to cause crashes or lag."`
	// LimitAllVelocityThreshold is the maximum allowed velocity in blocks per tick for all players. This is checked per-axis and not the overall velocity.
	LimitAllVelocityThreshold float32 `json:"limit_all_velocity_threshold" comment:"The maximum allowed velocity in blocks per tick for all players. This is checked per-axis and not the overall velocity."`
}

func Movement() MovementOpts {
	return Global.Movement
}
