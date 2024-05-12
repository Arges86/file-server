package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"file-server/database"
	"file-server/parser"
)

var db *database.Files

type Response struct {
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"statusCode"`
}

func main() {

	var err error
	db, err = database.New()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /files", getNames)
	mux.HandleFunc("POST /files", addFile)
	mux.HandleFunc("GET /files/{name}", getOneFile)

	http.ListenAndServe("localhost:4200", mux)

}

func getNames(w http.ResponseWriter, r *http.Request) {

	files, err := db.GetAll()
	if err != nil {
		fmt.Println(err)
	}

	bytes, _ := json.MarshalIndent(files, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func addFile(w http.ResponseWriter, r *http.Request) {
	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		responseMessage(w, "Error Retrieving the File", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	fileParts := strings.Split(handler.Filename, ".")

	// if file isn't a CSV, reject it
	if fileParts[1] != "csv" {
		responseMessage(w, "file must be a csv file to uplaod", http.StatusUnsupportedMediaType)
		return
	}

	// parses input into json string
	data, err := parser.ReadAndParseCsv(file)
	if err != nil {
		responseMessage(w, fmt.Sprintf("error while handling csv file: %s", err), http.StatusInternalServerError)
		return
	}
	json, err := parser.CsvToJson(data)
	if err != nil {
		responseMessage(w, fmt.Sprintf("error while converting csv to json file: %s", err), http.StatusInternalServerError)
		return
	}

	// writes to sqlite table
	resp, err := db.SaveFile(json, fileParts[0])
	if err != nil {
		fmt.Println(err)
		responseMessage(w, "error saving file", http.StatusInternalServerError)
		return
	}

	responseMessage(w, fmt.Sprintf("file %d saved", resp), http.StatusCreated)
}

func getOneFile(w http.ResponseWriter, r *http.Request) {

	name := r.PathValue("name")
	file, err := db.GetFile(name)
	if err != nil {
		responseMessage(w, fmt.Sprintf("%s not found", name), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(file))
}

func responseMessage(w http.ResponseWriter, message string, status int) {
	response := Response{
		Message:    message,
		StatusCode: status,
	}
	bytes, _ := json.MarshalIndent(response, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write(bytes)
}
