package api

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/mannulus-immortalis/xmtask/internal/models"
)

type api struct {
	log    *zerolog.Logger
	stor   models.StorageInt
	auth   models.AuthInt
	notify models.NotifyInt
	r      *gin.Engine
	srv    *http.Server
}

func New(log *zerolog.Logger, stor models.StorageInt, auth models.AuthInt, notify models.NotifyInt) *api {
	a := api{
		log:    log,
		stor:   stor,
		auth:   auth,
		notify: notify,
		r:      gin.New(),
	}
	a.SetupRoutes()
	return &a
}

func (a *api) Run(addr string) error {
	a.srv = &http.Server{
		Addr:    addr,
		Handler: a.r,
	}
	return a.srv.ListenAndServe()
}

func (a *api) Close() {
	_ = a.srv.Close()
}

func (a *api) SetupRoutes() {
	a.r.Use(gin.Recovery())
	a.r.Use(corsMiddleware())
	a.r.GET("/alive", a.Alive)

	a.r.POST("/api/v1/company", a.RequireRole(models.RoleWriter), a.CreateItem)
	a.r.PATCH("/api/v1/company/:id", a.RequireRole(models.RoleWriter), a.UpdateItem)
	a.r.DELETE("/api/v1/company/:id", a.RequireRole(models.RoleWriter), a.DeleteItem)
	a.r.GET("/api/v1/company/:id", a.RequireRole(models.RoleReader), a.GetItem)
}

func (a *api) AbortWithError(ctx *gin.Context, code int, err error) {
	e := models.ErrorResponse{Error: err.Error()}
	ctx.AbortWithStatusJSON(code, e)
}

func (a *api) RequireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		h := ctx.GetHeader("Authorization")
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			a.log.Error().Msg("Authorization header is invalid")
			a.AbortWithError(ctx, http.StatusForbidden, models.ErrJWTInvalid)
			return
		}
		hasRole, err := a.auth.TokenHasRole(parts[1], role)
		if err != nil {
			a.log.Err(err).Msg("Authorization check failed")
			a.AbortWithError(ctx, http.StatusForbidden, models.ErrJWTInvalid)
			return
		}
		if !hasRole {
			a.log.Err(err).Msg("Access denied")
			a.AbortWithError(ctx, http.StatusForbidden, models.ErrJWTRoleMissing)
			return
		}

		ctx.Next()
	}
}

func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"ResponseType",
			"accept",
			"origin",
			"Cache-Control",
			"X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
}
