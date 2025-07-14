package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func getFeatureHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
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

	var value bool
	err := db.QueryRow("SELECT value FROM features WHERE name = ?", name).Scan(&value)
	if err == sql.ErrNoRows {
		http.Error(w, "Feature not found", http.StatusInternalServerError)
		return
	} else if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := Feature{Name: name, Value: value}
	json.NewEncoder(w).Encode(response)
}
func listFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, value FROM features")
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var feature Feature
		if err := rows.Scan(&feature.Id, &feature.Name, &feature.Value); err != nil {
			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		features = append(features, feature)
	}

	json.NewEncoder(w).Encode(features)
}
func updateFeatureHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT allowed", http.StatusMethodNotAllowed)
		return
	}

	var feature Feature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, "Invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if feature.Name == "" {
		http.Error(w, "Feature name is required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE features SET value = ? WHERE name = ?", feature.Value, feature.Name)
	if err != nil {
		http.Error(w, "DB update error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
