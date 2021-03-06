package sessions

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mssola/user_agent"

	"github.com/gin-gonic/gin"
	"github.com/machinable/machinable/auth"
	"github.com/machinable/machinable/config"
	"github.com/machinable/machinable/dsi/interfaces"
	"github.com/machinable/machinable/dsi/models"
	as "github.com/machinable/machinable/sessions"
)

// New returns a pointer to a new `Users` struct
func New(db interfaces.Datastore, config *config.AppConfig) *Sessions {
	return &Sessions{
		store:  db,
		config: config,
		jwt:    auth.NewJWT(config),
	}
}

// Sessions wraps the datastore and any HTTP handlers for project user sessions
type Sessions struct {
	store  interfaces.Datastore
	config *config.AppConfig
	jwt    *auth.JWT
}

func (s *Sessions) generateSession(userID, ip, userAgent string) *models.Session {
	location, _ := as.GetGeoIP(ip, s.config.IPStackKey)

	ua := user_agent.New(userAgent)

	bname, bversion := ua.Browser()
	session := &models.Session{
		UserID:       userID,
		Location:     location,
		Mobile:       ua.Mobile(),
		IP:           ip,
		LastAccessed: time.Now(),
		Browser:      bname + " " + bversion,
		OS:           ua.OS(),
	}

	return session
}

// CreateSession creates a new project user session
func (s *Sessions) CreateSession(c *gin.Context) {
	projectSlug := c.MustGet("project").(string)

	// basic auth for login
	authorizationHeader, _ := c.Request.Header["Authorization"]
	if len(authorizationHeader) <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "no authorization header"})
		return
	}
	authzHeader := strings.SplitN(authorizationHeader[0], " ", 2)

	if len(authzHeader) != 2 || strings.ToLower(authzHeader[0]) != "basic" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "malformed authorization header"})
		return
	}

	payload, _ := base64.StdEncoding.DecodeString(authzHeader[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "no authorization header payload"})
		return
	}

	userName := pair[0]
	userPassword := strings.Trim(pair[1], "\n")

	if userName == "" {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}

	// find project
	project, err := s.store.GetProjectBySlug(projectSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project does not exist"})
		return
	}

	log.Println(project.ID)
	log.Println(userName)
	user, err := s.store.GetUserByUsername(project.ID, userName)
	log.Println(user)
	log.Println(err)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "username not found"})
		return
	}

	// compare passwords
	if !auth.CompareHashAndPassword(user.PasswordHash, userPassword) {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid password"})
		return
	}

	// create access token
	claims := jwt.MapClaims{
		"projects": map[string]interface{}{
			projectSlug: true,
		},
		"user": map[string]interface{}{
			"id":     user.ID,
			"name":   user.Username,
			"active": true,
			"read":   user.Read,
			"write":  user.Write,
			"role":   user.Role,
			"type":   "project",
		},
	}

	accessToken, err := s.jwt.CreateAccessToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create the access token"})
		return
	}

	// create session in database (refresh token)
	session := s.generateSession(user.ID, c.ClientIP(), c.Request.UserAgent())
	err = s.store.CreateSession(project.ID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	refreshToken, err := s.jwt.CreateRefreshToken(session.ID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create refresh token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Successfully logged in",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"session_id":    session.ID,
	})
}

// ListSessions lists all active user sessions for a project
func (s *Sessions) ListSessions(c *gin.Context) {
	projectID := c.MustGet("projectId").(string)

	sessions, err := s.store.ListSessions(projectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

// RevokeSession deletes a session from the project collection
func (s *Sessions) RevokeSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	projectID := c.MustGet("projectId").(string)

	err := s.store.DeleteSession(projectID, sessionID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// RefreshSession uses the refresh token to generate a new access token
func (s *Sessions) RefreshSession(c *gin.Context) {
	projectID := c.MustGet("projectId").(string)
	projectSlug := c.MustGet("project").(string)
	// get session and user id from context, should have been injected by ValidateRefreshToken
	sessionID, ok := c.MustGet("session_id").(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no session"})
		return
	}

	userID, ok := c.MustGet("user_id").(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no user"})
		return
	}

	// load session to update last accessed

	// verify session exists
	_, err := s.store.GetSession(projectID, sessionID)

	if err != nil {
		// no documents in result, user does not exist
		c.JSON(http.StatusNotFound, gin.H{"message": "error creating access token."})
		return
	}

	// verify user exists
	user, err := s.store.GetUserByID(projectID, userID)
	if err != nil {
		// no documents in result, user does not exist
		c.JSON(http.StatusNotFound, gin.H{"message": "error creating access token."})
		return
	}

	// create access token
	claims := jwt.MapClaims{
		"projects": map[string]interface{}{
			projectSlug: true,
		},
		"user": map[string]interface{}{
			"id":     user.ID,
			"name":   user.Username,
			"active": true,
			"read":   user.Read,
			"write":  user.Write,
			"role":   user.Role,
			"type":   "project",
		},
	}

	accessToken, err := s.jwt.CreateAccessToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create the access token"})
		return
	}

	// update session `last_accessed` time
	err = s.store.UpdateProjectSessionLastAccessed(projectID, sessionID, time.Now())

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}
