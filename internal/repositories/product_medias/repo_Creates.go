package product_medias

import (
	"context"
	wsqlx "github.com/SyaibanAhmadRamadhan/sqlx-wrapper"
	"github.com/mini-e-commerce-microservice/product-service/internal/model"
	"github.com/mini-e-commerce-microservice/product-service/internal/util/tracer"
)

func (r *repository) Creates(ctx context.Context, input CreatesInput) (err error) {
	if input.Data == nil && len(input.Data) <= 0 {
		return
	}

	query := r.sq.Insert("product_medias").Columns(
		"product_id", "media", "media_type", "is_primary_media",
	)
	for _, datum := range input.Data {
		query = query.Values(datum.ProductID, datum.Media, datum.MediaType, datum.IsPrimaryMedia)
	}

	rdbms := input.Tx
	if input.Tx == nil {
		rdbms = r.rdbms
	}

	_, err = rdbms.ExecSq(ctx, query)
	if err != nil {
		return tracer.Error(err)
	}
	return
}

type CreatesInput struct {
	Data []model.ProductMedia
	Tx   wsqlx.WriterCommand
}
