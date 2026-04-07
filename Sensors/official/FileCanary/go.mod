module github.com/honeywire/sensors/filecanary

go 1.25.0

require (
	github.com/fsnotify/fsnotify v1.9.0
	github.com/honeywire/sdk-go v0.0.0-00010101000000-000000000000
)

require golang.org/x/sys v0.42.0 // indirect

replace github.com/honeywire/sdk-go => ../../../SDKs/go-honeywire
