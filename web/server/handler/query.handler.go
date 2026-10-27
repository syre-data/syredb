package handler

import (
	"net/http"
	"syredb/database"
	"syredb/query"
	"syredb/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type QueryHandler struct {
	db            *database.DBConnection
	query_service *service.QueryService
}

func NewQueryHandler(
	db *database.DBConnection,
	query_service *service.QueryService,
) *QueryHandler {
	return &QueryHandler{
		db:            db,
		query_service: query_service,
	}
}

func (h *QueryHandler) DataSearch(c *echo.Context) error {
	type dataSearchQuery struct {
		Query   string
		Project uuid.UUID
	}

	user_id := c.Get(UserIdKey).(uuid.UUID)

	var searchQuery dataSearchQuery
	err := c.Bind(&searchQuery)
	if err != nil {
		c.Logger().With(
			"user", user_id,
		).Error("could not bind data search query")
		return c.NoContent(http.StatusBadRequest)
	}

	exprs, err := query.Parse(searchQuery.Query)
	if err != nil {
		c.Logger().With(
			"error", err,
			"user", user_id,
			"query", searchQuery.Query,
		).Error("could not parse serach query")
		return c.NoContent(http.StatusBadRequest)
	}

	var data []uuid.UUID
	if searchQuery.Project == uuid.Nil {
		data, err = h.query_service.SearchData(user_id, exprs)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user_id,
				"query", searchQuery.Query,
			).Error("could not query data")
			return c.NoContent(http.StatusInternalServerError)
		}
	} else {
		data, err = h.query_service.SearchProjectData(user_id, searchQuery.Project, exprs)
		if err != nil {
			c.Logger().With(
				"error", err,
				"user", user_id,
				"project", searchQuery.Project,
				"query", searchQuery.Query,
			).Error("could not query project data")
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, data)

}
