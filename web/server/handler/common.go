package handler

import (
	"net/url"

	"github.com/google/uuid"
)

type ResourceCreated struct {
	Id uuid.UUID `json:"id"`
}

func (r ResourceCreated) ToUrlQuery() url.Values {
	loc_q := url.Values{}
	loc_q.Set("id", r.Id.String())
	return loc_q
}

type ResourceCreatedMultiple struct {
	Ids []uuid.UUID `json:"ids"`
}
