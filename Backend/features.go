package main

import (
	"encoding/json"
	"net/http"
	
	sharedDB "github.com/hirepilot/shared/db"
	"github.com/hirepilot/shared/models"
)

func getFeatureHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Feature name is required", http.StatusBadRequest)
		return
	}

	value, err := sharedDB.GetFeatureValue(name)
	if err == sharedDB.ErrNotFound {
		http.Error(w, "Feature not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.Feature{Name: name, Value: value}
	json.NewEncoder(w).Encode(response)
}
func listFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "GET, OPTIONS")
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	features, err := sharedDB.GetAllFeatures()
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(features)
}
func updateFeatureHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		handleCORS(w, "PUT, OPTIONS")
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
		return
	}

	var feature models.Feature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if feature.Name == "" {
		http.Error(w, "Feature name is required", http.StatusBadRequest)
		return
	}

	err := sharedDB.UpdateFeatureValue(feature.Name, feature.Value)
	if err != nil {
		http.Error(w, "DB update error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
