module github.com/ui-sound-system/uss-go/examples/oto

go 1.26.0

require (
	github.com/ebitengine/oto/v3 v3.5.1
	github.com/ui-sound-system/uss-go v0.0.0-20261025085602-68874306c1c3
)

require (
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/jfreymuth/pulse v0.1.3 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

// Use the enclosing checkout so this example includes unpublished USS changes.
replace github.com/ui-sound-system/uss-go => ../..
