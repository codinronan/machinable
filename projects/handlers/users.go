package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"

	"bitbucket.org/nsjostrom/machinable/projects/database"
	"bitbucket.org/nsjostrom/machinable/projects/models"
	"github.com/gin-gonic/gin"
)

// AddUser creates a new user for this project
func AddUser(c *gin.Context) {
	var newUser models.NewProjectUser
	projectSlug := c.MustGet("project").(string)

	c.BindJSON(&newUser)

	user := &models.ProjectUser{
		ID:           objectid.New(), // I don't like this
		Created:      time.Now(),
		PasswordHash: newUser.Password, // salt and hash
		Username:     newUser.Username,
		Read:         newUser.Read,
		Write:        newUser.Write,
	}

	// Get the resources.{resourcePathName} collection
	rc := database.Collection(database.UserDocs(projectSlug))
	_, err := rc.InsertOne(
		context.Background(),
		user,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// ListUsers lists all users of this project
func ListUsers(c *gin.Context) {
	projectSlug := c.MustGet("project").(string)
	users := make([]*models.ProjectUser, 0)

	collection := database.Connect().Collection(database.UserDocs(projectSlug))

	cursor, err := collection.Find(
		context.Background(),
		bson.NewDocument(),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for cursor.Next(context.Background()) {
		var user models.ProjectUser
		err := cursor.Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, &user)
	}

	c.JSON(http.StatusOK, gin.H{"items": users})
}

// GetUser retrieves a single user of this project by ID
func GetUser(c *gin.Context) {
	//projectSlug := c.MustGet("project").(string)
	c.JSON(http.StatusNotImplemented, gin.H{})
}

// DeleteUser removes a user by ID
func DeleteUser(c *gin.Context) {
	//projectSlug := c.MustGet("project").(string)
	c.JSON(http.StatusNotImplemented, gin.H{})
}