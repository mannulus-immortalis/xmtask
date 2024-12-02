package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"unicode/utf8"

	"github.com/mannulus-immortalis/xmtask/internal/models"
)

func (a *api) Alive(ctx *gin.Context) {
	ctx.String(http.StatusOK, "ok")
}

func (a *api) CreateItem(ctx *gin.Context) {
	var req models.ItemCreateRequest
	err := ctx.BindJSON(&req)
	if err != nil {
		a.log.Err(err).Msg("invalid create request")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidRequest)
		return
	}

	if req.Name == "" || utf8.RuneCountInString(req.Name) > 15 {
		a.log.Error().Msg("invalid name")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidName)
		return
	}

	if utf8.RuneCountInString(req.Description) > 3000 {
		a.log.Error().Msg("invalid description")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidDescription)
		return
	}

	if _, ok := models.AcceptableLegalTypes[req.Type]; !ok {
		a.log.Error().Str("Type", req.Type).Msg("invalid type")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidType)
		return
	}

	id, err := a.stor.CreateItem(ctx, &req)
	if err != nil {
		a.log.Err(err).Msg("db create request failed")
		switch err {
		case models.ErrDuplicateName:
			a.AbortWithError(ctx, http.StatusBadRequest, models.ErrDuplicateName)
		default:
			a.AbortWithError(ctx, http.StatusInternalServerError, models.ErrDBError)
		}
		return
	}
	item := models.ItemResponse{
		ID:            *id,
		Name:          req.Name,
		Description:   req.Description,
		EmployeeCount: req.EmployeeCount,
		IsRegistered:  req.IsRegistered,
		Type:          req.Type,
	}

	err = a.notify.Send(models.EventNotifications{ID: *id, Event: models.EventTypeCreated})
	if err != nil {
		a.log.Err(err).Msg("notification send failed")
	}

	ctx.JSON(http.StatusCreated, item)
}

func (a *api) UpdateItem(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.log.Err(err).Msg("invalid id")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidID)
		return
	}

	var req models.ItemUpdateRequest
	err = ctx.BindJSON(&req)
	if err != nil {
		a.log.Err(err).Msg("invalid update request")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidRequest)
		return
	}

	if req.Name == nil && req.Description == nil && req.EmployeeCount == nil && req.IsRegistered == nil && req.Type == nil {
		a.log.Error().Msg("empty update request")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrNothingToDo)
		return
	}

	if req.Name != nil && (*req.Name == "" || utf8.RuneCountInString(*req.Name) > 15) {
		a.log.Error().Msg("invalid name")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidName)
		return
	}

	if req.Description != nil && utf8.RuneCountInString(*req.Description) > 3000 {
		a.log.Error().Msg("invalid description")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidDescription)
		return
	}

	if req.Type != nil {
		if _, ok := models.AcceptableLegalTypes[*req.Type]; !ok {
			a.log.Error().Str("Type", *req.Type).Msg("invalid type")
			a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidType)
			return
		}
	}

	err = a.stor.UpdateItem(ctx, id, &req)
	if err != nil {
		a.log.Err(err).Str("ID", id.String()).Msg("db update request failed")
		switch err {
		case models.ErrNotFound:
			a.AbortWithError(ctx, http.StatusNotFound, models.ErrNotFound)
		case models.ErrDuplicateName:
			a.AbortWithError(ctx, http.StatusBadRequest, models.ErrDuplicateName)
		default:
			a.AbortWithError(ctx, http.StatusInternalServerError, models.ErrDBError)
		}
		return
	}

	err = a.notify.Send(models.EventNotifications{ID: id, Event: models.EventTypeUpdated})
	if err != nil {
		a.log.Err(err).Msg("notification send failed")
	}

	ctx.Status(http.StatusOK)
}

func (a *api) DeleteItem(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.log.Err(err).Msg("invalid id")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidID)
		return
	}

	err = a.stor.DeleteItem(ctx, id)
	if err != nil {
		a.log.Err(err).Str("ID", id.String()).Msg("db delete request failed")
		switch err {
		case models.ErrNotFound:
			a.AbortWithError(ctx, http.StatusNotFound, models.ErrNotFound)
		default:
			a.AbortWithError(ctx, http.StatusInternalServerError, models.ErrDBError)
		}
		return
	}

	err = a.notify.Send(models.EventNotifications{ID: id, Event: models.EventTypeDeleted})
	if err != nil {
		a.log.Err(err).Msg("notification send failed")
	}

	ctx.Status(http.StatusOK)
}

func (a *api) GetItem(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		a.log.Err(err).Msg("invalid id")
		a.AbortWithError(ctx, http.StatusBadRequest, models.ErrInvalidID)
		return
	}

	item, err := a.stor.GetItem(ctx, id)
	if err != nil {
		a.log.Err(err).Str("ID", id.String()).Msg("db select request failed")
		switch err {
		case models.ErrNotFound:
			a.AbortWithError(ctx, http.StatusNotFound, models.ErrNotFound)
		default:
			a.AbortWithError(ctx, http.StatusInternalServerError, models.ErrDBError)
		}
		return
	}

	ctx.JSON(http.StatusOK, item)
}
