package resources

import (
	"bitbucket.org/nsjostrom/machinable/dsi/interfaces"
	"bitbucket.org/nsjostrom/machinable/middleware"
	"github.com/gin-gonic/gin"
)

// SetRoutes sets all of the appropriate routes to handlers for project collections
func SetRoutes(engine *gin.Engine, datastore interfaces.ResourcesDatastore) error {
	// create new Resources handler with datastore
	handler := New(datastore)

	// admin/mgmt routes
	// Only application users have access to resource definitions
	resources := engine.Group("/resources")
	resources.Use(middleware.AppUserJwtAuthzMiddleware())
	resources.Use(middleware.AppUserProjectAuthzMiddleware())

	resources.POST("/", handler.AddResourceDefinition)
	resources.GET("/", handler.ListResourceDefinitions)
	resources.GET("/:resourceDefinitionID", handler.GetResourceDefinition)
	resources.DELETE("/:resourceDefinitionID", handler.DeleteResourceDefinition)

	return nil
}