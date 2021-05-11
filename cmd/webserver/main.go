package main

import (
	"ethstats/ethtools"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// func rootHandler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
// }

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
	db := ethtools.NewDatabaseConnection(ethtools.GetConfig())
	entry, found := ethtools.DbGetAnalysisById(db, id)
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
	config := ethtools.GetConfig()
	listenAddr := fmt.Sprintf("%s:%d", config.WebserverHost, config.WebserverPort)

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)
	e.GET("/analysis/:id", getAnalysis)

	// Start server
	e.Logger.Fatal(e.Start(listenAddr))

	// // Register routes
	// http.HandleFunc("/", rootHandler)
	// http.HandleFunc("/analysis", getAnalysis)

	// // Start webserver

	// logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	// logger.Println("Server is starting on", listenAddr)
	// log.Fatal(http.ListenAndServe(listenAddr, nil))
}
