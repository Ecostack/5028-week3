package main

import (
	"os"
	"testing"
)

func TestParseResponse(t *testing.T) {
	output, err := os.ReadFile("example_api_output.html")
	if err != nil {
		t.Error("Error reading file")
	}
	movies, err := parseAPIResponse(string(output))
	if err != nil {
		t.Error("Error parsing response")
	}
	if len(movies) != 20 {
		t.Fatal("Expected 20 movies, got", len(movies))
	}
	if movies[0].Title != "Bird Box" {
		t.Error("Expected Bird Box, got", movies[0].Title)
	}
}

func TestParseAndSave(t *testing.T) {
	db := mustSetupDB()
	clearDB(db)
	output, err := os.ReadFile("example_api_output.html")
	if err != nil {
		t.Error("Error reading file")
	}
	data, err := parseAPIResponse(string(output))
	if err != nil {
		t.Error("Error parsing response")
	}
	for _, movie := range data {
		db.Create(movie)
	}
	var movies []Movie

	db.Find(&movies)
	if len(movies) != 20 {
		t.Fatal("Expected 20 movies, got", len(movies))
	}
}
