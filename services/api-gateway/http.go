package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {
	//if r.Method != http.MethodPost {
	//
	//}
	//
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)

	if reqBody.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, "test response")
	// TODO: Call trip service
}
