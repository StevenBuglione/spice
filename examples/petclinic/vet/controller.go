package vet

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/StevenBuglione/spice/view"
	"github.com/StevenBuglione/spice/web"
)

// @import { Controller, Get } from "github.com/StevenBuglione/spice/annotation/web"

const vetPageSize = 5

// Controller serves veterinarian browser and API representations.
//
// @Controller
type Controller struct {
	vets Repository
}

// NewController constructs the veterinarian HTTP boundary.
func NewController(vets Repository) (*Controller, error) {
	if vets == nil {
		return nil, errors.New(
			"construct veterinarian controller: repository is nil",
		)
	}
	return &Controller{vets: vets}, nil
}

// ListHTML renders one veterinarian page.
//
// @Get("/vets.html")
func (controller *Controller) ListHTML(
	ctx context.Context,
	request ListRequest,
) (view.Result, error) {
	page := request.Page
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return view.Result{}, invalidPage(
			"the page must be positive",
			fmt.Errorf("page %d is not positive", page),
		)
	}
	values, total, err := controller.vets.FindPage(
		ctx,
		(page-1)*vetPageSize,
		vetPageSize,
	)
	if err != nil {
		return view.Result{}, fmt.Errorf("list veterinarian page: %w", err)
	}
	totalPages := max(1, (total+vetPageSize-1)/vetPageSize)
	if page > totalPages {
		return view.Result{}, invalidPage(
			"the requested page does not exist",
			fmt.Errorf("page %d exceeds %d", page, totalPages),
		)
	}
	return view.Render("vets/vetList", ListModel{
		Vets:         values,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalItems:   total,
		PreviousPage: page - 1,
		NextPage:     page + 1,
		HasPrevious:  page > 1,
		HasNext:      page < totalPages,
	})
}

// ListJSON returns every veterinarian in stable display order.
//
// @Get("/vets")
func (controller *Controller) ListJSON(
	ctx context.Context,
	_ AllRequest,
) (Vets, error) {
	values, err := controller.vets.FindAll(ctx)
	if err != nil {
		return Vets{}, fmt.Errorf("list veterinarians: %w", err)
	}
	return Vets{VetList: values}, nil
}

func invalidPage(detail string, cause error) error {
	return web.NewError(web.Problem{
		Type:   "https://spice.dev/problems/petclinic-vet-page",
		Title:  "Invalid veterinarian page",
		Status: http.StatusBadRequest,
		Detail: detail,
	}, cause)
}
