package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/metachris/ethereum-go-experiments/config"
	"github.com/metachris/ethereum-go-experiments/ethstats"
)

func getAnalysis(c echo.Context) (err error) {
	// Grab 'id' URL parameter
	idStr := c.Param("id")
	if len(idStr) < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Convert id to int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get entry from DB
	db := ethstats.GetDatabase(config.GetConfig().Database)
	entry, found := ethstats.DbGetAnalysisById(db, id)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// return
	return c.JSON(http.StatusOK, entry)
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func main() {
	config := config.GetConfig()
	listenAddr := fmt.Sprintf("%s:%d", config.WebserverHost, config.WebserverPort)

	// Echo instance
	e := echo.New()

	// Middleware
	// e.Use(middleware.Logger()) // JSON logging
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status} t=${latency} in=${bytes_in}, out=${bytes_out}\n",
	}))

	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	e.GET("/api/analysis/:id", getAnalysis)

	// Start server
	e.Logger.Fatal(e.Start(listenAddr))
}
