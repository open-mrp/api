package domain

import (
	"time"

	"github.com/augno/api/shared/pagination"
)

type ProductType struct {
	ID        string
	Name      string `audit:"name"`
	Code      string `audit:"code"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListProductTypesParams struct {
	Cursor *string
	Limit  int32
	Query  *string
}

type ListProductTypesResult struct {
	ProductTypes []*ProductType
	PageInfo     pagination.PageInfo
}

type CreateProductTypeParams struct {
	Name string
	Code string
}

type UpdateProductTypeParams struct {
	ProductTypeID string
	Name          *string
	Code          *string
}
