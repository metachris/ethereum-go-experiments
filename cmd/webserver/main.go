package main

import (
	"encoding/json"
	"ethstats/ethtools"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func getAnalysis(w http.ResponseWriter, r *http.Request) {
	// fmt.Println(r.URL.Path, r.URL.Query())
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(ids[0])
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	db := ethtools.NewDatabaseConnection(ethtools.GetConfig())
	entry, found := ethtools.DbGetAnalysisById(db, id)
	if !found {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	fmt.Println(entry)
	err = json.NewEncoder(w).Encode(entry)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func main() {
	// Register routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/analysis", getAnalysis)

	// Start webserver
	config := ethtools.GetConfig()
	listenAddr := fmt.Sprintf("%s:%d", config.WebserverHost, config.WebserverPort)

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting on", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
