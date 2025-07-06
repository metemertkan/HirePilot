package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

func addPromptHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var prompt Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(
		"INSERT INTO prompts (name, prompt) VALUES (?, ?)",
		prompt.Name, prompt.Prompt,
	)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func listPromptsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Query("SELECT id, name, prompt FROM prompts")
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Prompt struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Prompt string `json:"prompt"`
	}

	var prompts []Prompt
	for rows.Next() {
		var p Prompt
		if err := rows.Scan(&p.ID, &p.Name, &p.Prompt); err != nil {
			http.Error(w, "DB scan error", http.StatusInternalServerError)
			return
		}
		prompts = append(prompts, p)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prompts); err != nil {
		http.Error(w, "JSON encode error", http.StatusInternalServerError)
		return
	}
}

func updatePromptHandler(w http.ResponseWriter, r *http.Request) {
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

	var prompt struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	res, err := db.Exec(
		"UPDATE prompts SET name = ?, prompt = ? WHERE id = ?",
		prompt.Name, prompt.Prompt, prompt.ID,
	)
	if err != nil {
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}
	n, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if n == 0 {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getPromptHandler(w http.ResponseWriter, r *http.Request) {
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

	// Parse ID from query parameters
	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Convert id to int
	id, err := strconv.Atoi(ids[0])
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	// Query the prompt by ID
	var p struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Prompt string `json:"prompt"`
	}
	err = db.QueryRow("SELECT id, name, prompt FROM prompts WHERE id = ?", id).Scan(&p.ID, &p.Name, &p.Prompt)
	if err == sql.ErrNoRows {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		http.Error(w, "JSON encode error", http.StatusInternalServerError)
		return
	}
}
