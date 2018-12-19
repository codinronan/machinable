package projects

import (
	"bitbucket.org/nsjostrom/machinable/dsi/interfaces"
	"bitbucket.org/nsjostrom/machinable/middleware"
	"github.com/gin-gonic/gin"
)

// SetRoutes sets all of the appropriate routes to handlers for projects
func SetRoutes(engine *gin.Engine, datastore interfaces.ProjectsDatastore) error {
	handler := New(datastore)

	// project endpoints
	projects := engine.Group("/projects")
	projects.Use(middleware.AppUserJwtAuthzMiddleware())
	projects.GET("/", handler.ListUserProjects)
	projects.POST("/", handler.CreateProject)
	projects.PUT("/:projectSlug", handler.UpdateProject)
	projects.DELETE("/:projectSlug", handler.DeleteUserProject)

	return nil
}
