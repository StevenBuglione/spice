// Package event defines canonical descriptors for typed application events.
package event

import "github.com/StevenBuglione/spice/annotation/sdk"

// Topic marks a provider function that declares one typed event topic.
//
// The topic's exact payload type and ownership come from the Go signature and
// application module. Generated publishing remains explicit and observable.
//
//	// @spice.import { Topic } from "github.com/StevenBuglione/spice/annotation/event"
//	// @Topic
//	func OrderChangedTopic() event.Topic[OrderChanged]
func Topic() sdk.Definition {
	return sdk.Definition{
		Name:    "event.Topic",
		Summary: "Declares a typed application event topic.",
		Targets: []sdk.Target{sdk.TargetFunction},
		Examples: []sdk.Example{{
			Title: "Typed topic",
			Code:  "// @Topic\nfunc OrderChangedTopic() event.Topic[OrderChanged]",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "event/topic",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "EventTopicHandler",
			},
		},
	}
}
