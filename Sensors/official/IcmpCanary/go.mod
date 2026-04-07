module github.com/honeywire/sensors/icmpcanary

go 1.25.0

replace github.com/honeywire/sdk-go => ../../../SDKs/go-honeywire

require (
	github.com/honeywire/sdk-go v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.52.0
)

require golang.org/x/sys v0.42.0 // indirect
