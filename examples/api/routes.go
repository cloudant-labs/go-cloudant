/**
 * routes
 * - define gin routes
 */

package main

// Routes provides all routes and route-specific middleware set-up
func (api *API) Routes() {

	// Define routes
	r := api.Server.Router
	r.GET("/doc/:id", api.Get)
	r.PUT("/doc/:id", api.Put)
}
