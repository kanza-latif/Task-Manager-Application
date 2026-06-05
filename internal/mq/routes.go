package mq

// Exchange and routing keys — must match the MDF RabbitMQ setup exactly
const (
	Exchange = "radius_exchange"

	// Session routing keys (consumed by MDF, not this app)
	RouteSessionStart = "session.start"
	RouteSessionStop  = "session.stop"
	RouteSessionFinal = "session.final"

	// Stats — task manager consumes this
	RouteSessionStats = "session.stats"

	// Bootstrap — task manager publishes these
	RouteCGNATLoad     = "bootstrap.cgnat"
	RouteWhitelistLoad = "bootstrap.whitelist"
)
