package handlers

import "github.com/gin-gonic/gin"

func Health(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) }
// Health checks the API's health status and returns a simple JSON response.
// It responds with a 200 status code and a JSON object indicating the service is operational.
// This endpoint can be used for basic uptime monitoring and health checks by load balancers or monitoring systems.