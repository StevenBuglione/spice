module example.com/spice-annotation-app

go 1.26.0

toolchain go1.26.5

tool (
	example.com/spice-annotation-fixture/cmd/spice-annotations
	github.com/StevenBuglione/spice/cmd/spice-annotation-core
)

require github.com/StevenBuglione/spice v0.0.0

require example.com/spice-annotation-fixture v0.0.0 // indirect

replace example.com/spice-annotation-fixture => ../annotationfixture

replace github.com/StevenBuglione/spice => ../..
