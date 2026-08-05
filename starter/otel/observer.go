package otel

import (
	"context"

	spiceevent "github.com/spice-framework/spice/event"
	"github.com/spice-framework/spice/web"
)

// Observer is the generated application-level OpenTelemetry adapter. It
// composes HTTP route and typed module-event telemetry without global state.
type Observer struct {
	http  *HTTPObserver
	event *EventObserver
}

// NewObserver constructs the complete generated application observer.
func NewObserver(options Options) (*Observer, error) {
	httpObserver, err := NewHTTPObserver(options)
	if err != nil {
		return nil, err
	}
	eventObserver, err := NewEventObserver(options)
	if err != nil {
		return nil, err
	}
	return &Observer{http: httpObserver, event: eventObserver}, nil
}

// BeginHTTP implements web.HTTPObserver.
func (observer *Observer) BeginHTTP(
	ctx context.Context,
	route web.RouteMetadata,
) (context.Context, func(web.HTTPResult)) {
	if observer == nil {
		return ctx, nil
	}
	return observer.http.BeginHTTP(ctx, route)
}

// BeginEvent implements event.Observer.
func (observer *Observer) BeginEvent(
	ctx context.Context,
	interaction spiceevent.Interaction,
) (context.Context, func(spiceevent.Result)) {
	if observer == nil {
		return ctx, nil
	}
	return observer.event.BeginEvent(ctx, interaction)
}

var (
	_ web.HTTPObserver    = (*Observer)(nil)
	_ spiceevent.Observer = (*Observer)(nil)
)
