/**
 * server
 * - create gin server
 */

package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Server is the object which ties together all the dependencies
type Server struct {
	Router *gin.Engine

	HTTPServer *http.Server
}

// NewServer creates a new server with the opts given.
func NewServer(options ...func(*Server)) *Server {

	log.Trace("cgh-server: New")

	// Set gin log level (trace/debug levels will set gin to debug mode, the rest to release mode)
	lvl := Getenv("LOG_LEVEL")
	if lvl == "trace" || lvl == "debug" {
		lvl = "debug"
	} else {
		lvl = "release"
	}
	gin.SetMode(lvl)

	log.Trace("server: Gin log level set to " + lvl)

	// Create gin server with our defaults
	engine := gin.New()
	s := &Server{
		Router: engine,
		HTTPServer: &http.Server{
			MaxHeaderBytes: 1 << 20,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
		},
	}

	// Global middleware; disable logs for /health route
	s.Router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/health"))

	// Public routes
	s.Router.GET("/health", health)

	// Error middleware;
	s.Router.Use(ErrorMiddleware)

	// Customize server using functional options
	for _, option := range options {
		option(s)
	}

	return s
}

func health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}

// Run starts GIN server on PORT
func (s *Server) Run() {
	// Listen and serve
	port := Getenv("PORT")
	if port == "" {
		port = ":3000"
	}

	server := s.HTTPServer
	if server.Handler == nil {
		server.Handler = s.Router
	}
	server.Addr = port
	server.ReadTimeout = 10 * time.Second
	server.WriteTimeout = 10 * time.Second
	server.MaxHeaderBytes = 1 << 20

	log.WithFields(log.Fields{"port": port}).Debug("server: GIN accepting connections")
	server.ListenAndServe()
}

// ErrorMiddleware provides common error response
func ErrorMiddleware(c *gin.Context) {
	// handlers
	c.Next()

	// errors
	if c.Errors != nil {
		err := c.Errors.Last()
		if err.Meta != nil {
			log.WithFields(log.Fields{"err": err}).Debug(err.Meta)
			c.JSON(int(err.Type), gin.H{"error": err.Meta})
		} else {
			log.WithFields(log.Fields{"err": err}).Debug(err.Error())
			c.JSON(int(err.Type), gin.H{"error": err.Error()})
		}
	}
}
