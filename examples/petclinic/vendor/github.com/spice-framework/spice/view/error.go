package view

import (
	"errors"
	"net/http"

	"github.com/spice-framework/spice/web"
)

// WriteError maps an application error to a safe problem and renders the
// configured HTML error template. Renderers without an error template retain
// the standard RFC 9457 JSON behavior.
func (renderer *Renderer) WriteError(
	writer http.ResponseWriter,
	request *http.Request,
	err error,
	mapper web.ErrorMapper,
) error {
	if renderer == nil {
		return errors.New("write HTML error: renderer is nil")
	}
	if request == nil {
		return errors.New("write HTML error: request is nil")
	}
	if mapper == nil {
		mapper = web.DefaultErrorMapper
	}
	problem := mapper(request.Context(), err)
	if problem.Validate() != nil {
		problem = web.DefaultErrorMapper(
			request.Context(),
			errors.New("invalid application error mapping"),
		)
	}
	if renderer.errorName == "" || renderer.errorModel == nil {
		return web.WriteProblem(writer, problem)
	}
	renderErr := renderer.Render(
		request.Context(),
		writer,
		renderer.errorName,
		problem.Status,
		renderer.errorModel(request, problem),
	)
	if renderErr == nil {
		return nil
	}
	if fallbackErr := web.WriteProblem(writer, problem); fallbackErr != nil {
		return errors.Join(renderErr, fallbackErr)
	}
	return nil
}
