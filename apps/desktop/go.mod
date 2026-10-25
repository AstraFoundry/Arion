module arion.local/desktop

go 1.27.1

require (
	arion.local/core v0.0.0
	arion.local/models v0.0.0
	github.com/AstraFoundry/republic v0.0.0-20261020130734-85875f48f5b9
	github.com/ebitengine/purego v0.11.1
	github.com/egoist/mygo v0.4.6
	github.com/gen2brain/malgo v0.11.26
	golang.org/x/image v0.46.0
)

require (
	github.com/go-text/typesetting v0.3.5 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace (
	arion.local/core => ../../packages/core
	arion.local/models => ../../models
	github.com/egoist/mygo => ../../third_party/mygo
)

tool github.com/egoist/mygo/cmd/mygo
