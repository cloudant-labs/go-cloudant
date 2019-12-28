/**
 * controller
 * - GET and PUT operations
 */

package main

import (
	"errors"
	"net/http"

	"github.com/barshociaj/go-cloudant"
	"github.com/gin-gonic/gin"
)

type success struct {
	ID string `json:"id"`
}

// Get retrieves doc from Cloudant
func (api *API) Get(c *gin.Context) {
	// Use database
	db, _ := api.Cloudant.Use(DBNAME)

	// Check valid ID
	id := c.Param("id")
	if id == "" {
		c.Error(errors.New("No ID")).SetType(http.StatusBadRequest).SetMeta(BADREQUEST)
		return
	}

	doc := Doc{}
	// Get doc from Cloudant view
	err := db.Get(id, cloudant.NewDocQuery(), &doc)

	if err != nil && err.Error() == CLOUDANT404ERROR {
		c.Error(err).SetType(http.StatusNotFound).SetMeta(NOTFOUND)
		return
	}
	if err != nil {
		c.Error(err).SetType(http.StatusInternalServerError).SetMeta(INTERNALSERVERERROR)
		return
	}
	// Empty _id and _rev
	doc.ID = ""
	doc.Rev = ""
	c.JSON(http.StatusOK, doc)
}

// Put puts new doc into Cloudant
func (api *API) Put(c *gin.Context) {
	// Validate input
	var newDoc Doc
	if err := c.ShouldBindJSON(&newDoc); err != nil {
		c.Error(err).SetType(http.StatusBadRequest).SetMeta("Invalid document")
		return
	}

	// Use database
	db, _ := api.Cloudant.Use(DBNAME)

	// Check valid ID
	id := c.Param("id")
	if id == "" {
		c.Error(errors.New("No ID")).SetType(http.StatusBadRequest).SetMeta("No ID in the request")
		return
	}

	// Retrieve old doc
	oldDoc := Doc{}
	err := db.Get(id, cloudant.NewDocQuery(), oldDoc)

	if err != nil && err.Error() != CLOUDANT404ERROR {
		c.Error(err).SetType(http.StatusInternalServerError).SetMeta(INTERNALSERVERERROR)
		return
	}

	// Save in Cloudant
	if res, err := db.Insert(newDoc); err == nil {
		c.JSON(http.StatusOK, gin.H{"data": success{
			ID: res.ID,
		}})
		return
	}

	// Save failed
	c.Error(err).SetType(http.StatusInternalServerError).SetMeta("controller: Saving error")
}
