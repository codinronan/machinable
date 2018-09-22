package handlers

import (
	"context"
	"fmt"
	"net/http"

	"bitbucket.org/nsjostrom/machinable/database"

	"github.com/gin-gonic/gin"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// AddObject creates a new document of the resource definition
func AddObject(c *gin.Context) {
	resourcePathName := c.Param("resourcePathName")
	fieldValues := make(map[string]interface{})

	c.BindJSON(&fieldValues)

	// Get field definitions for this resource
	resourceDefinition, err := getDefinition(resourcePathName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create document for this resource based on the field definitions
	objectDocument, err := createFieldDocument(fieldValues, resourceDefinition.Fields)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the resources.{resourcePathName} collection
	rc := database.Collection(fmt.Sprintf(database.ResourceFormat, resourcePathName))
	result, err := rc.InsertOne(
		context.Background(),
		objectDocument,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set the inserted ID for the response
	fieldValues["id"] = result.InsertedID.(objectid.ObjectID).Hex()

	c.JSON(http.StatusCreated, fieldValues)
}

// ListObjects returns the list of objects for a resource
func ListObjects(c *gin.Context) {
	resourcePathName := c.Param("resourcePathName")
	collection := database.Collection(fmt.Sprintf(database.ResourceFormat, resourcePathName))
	response := make([]map[string]interface{}, 0)

	// Find the resource definition for this object
	resourceDefinition, err := getDefinition(resourcePathName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find all objects for this resource
	cursor, err := collection.Find(
		context.Background(),
		bson.NewDocument(),
	)

	// Create response from documents
	doc := bson.NewDocument()
	for cursor.Next(context.Background()) {
		doc.Reset()
		err := cursor.Decode(doc)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		fields, err := parseDocumentToMap(doc, resourceDefinition.Fields)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response = append(response, fields)
	}
	c.JSON(http.StatusOK, gin.H{"items": response, "definition": resourceDefinition})
}

// GetObject returns a single object with the resourceID for this resource
func GetObject(c *gin.Context) {
	resourcePathName := c.Param("resourcePathName")
	resourceID := c.Param("resourceID")
	collection := database.Collection(fmt.Sprintf(database.ResourceFormat, resourcePathName))

	// Find the resource definition for this object
	resourceDefinition, err := getDefinition(resourcePathName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create object ID from resource ID string
	objectID, err := objectid.FromHex(resourceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find object based on ID
	documentResult := collection.FindOne(
		nil,
		bson.NewDocument(
			bson.EC.ObjectID("_id", objectID),
		),
		nil,
	)

	if documentResult == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no documents for resource"})
	}

	// Decode result into document
	doc := bson.Document{}
	documentResult.Decode(&doc)
	// Lookup  definitions for this resource
	object, err := parseDocumentToMap(&doc, resourceDefinition.Fields)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, object)
}
