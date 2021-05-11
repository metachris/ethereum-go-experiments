package main

import (
	"encoding/json"
	"ethstats/ethtools"
	"fmt"
	"log"
	"net/http"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func getAnalysis(w http.ResponseWriter, r *http.Request) {
	db := ethtools.NewDatabaseConnection(ethtools.GetConfig())
	entry, found := ethtools.DbGetAnalysisById(db, 1)
	if !found {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println(entry)
	json.NewEncoder(w).Encode(entry)
}

func main() {
	// Register routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/analysis", getAnalysis)

	// Start webserver
	port := ethtools.GetConfig().WebserverPort
	fmt.Println("Webserver listening on port", port)
	listenAt := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(listenAt, nil))

}
