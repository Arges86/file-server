package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"file-server/config"
	"file-server/database"
	"file-server/parser"

	"github.com/tidwall/gjson"
)

var db *database.Database

type Response struct {
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"statusCode"`
}

const ct = "Content-Type"
const aj = "application/json"

func main() {

	var err error

	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err = database.New(conf.Database.File)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /files", getNames)
	mux.HandleFunc("POST /files", addFile)
	mux.HandleFunc("GET /files/{name}", getOneFile)

	addr := fmt.Sprintf("%s:%s", conf.Server.Host, conf.Server.Port)

	fmt.Printf("Server started: %s \n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		if err == http.ErrServerClosed {
			// Normal interrupt operation, ignore
		} else {
			log.Fatalf("Server failed to start due to err: %v", err)
		}
	}
}

func getNames(w http.ResponseWriter, r *http.Request) {

	files, err := db.GetAll()
	if err != nil {
		fmt.Println(err)
		responseMessage(w, "Error retrieving files", http.StatusInternalServerError)
		return
	}

	bytes, _ := json.MarshalIndent(files, "", "  ")
	w.Header().Set(ct, aj)
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
}

func addFile(w http.ResponseWriter, r *http.Request) {
	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println("Error Retrieving the file")
		fmt.Println(err)
		responseMessage(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	fileParts := strings.Split(handler.Filename, ".")

	// if file isn't a CSV, reject it
	if fileParts[1] != "csv" {
		responseMessage(w, "file must be a csv file to upload", http.StatusUnsupportedMediaType)
		return
	}

	separator := r.FormValue("separator")
	comma := getComma(separator)

	// parses input into json string
	data, err := parser.ReadAndParseCsv(file, comma)
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
	values := r.URL.Query()

	// gets 'any' query parameter as a key value pair
	var key, value string
	if len(values) > 0 {
		for k, v := range values {
			key = k
			value = v[0]
		}
	}

	file, err := db.GetFile(name)
	if err != nil {
		responseMessage(w, fmt.Sprintf("%s not found", name), http.StatusNotFound)
		return
	}

	file = filterResponse(file, key, value)

	w.Header().Set(ct, aj)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(file))
}

func responseMessage(w http.ResponseWriter, message string, status int) {
	response := Response{
		Message:    message,
		StatusCode: status,
	}
	bytes, _ := json.MarshalIndent(response, "", "  ")
	w.Header().Set(ct, aj)
	w.WriteHeader(status)
	w.Write(bytes)
}

// if `key` is present, filters file based on key/value pair
func filterResponse(file, key, value string) string {
	if len(key) > 0 {
		path := fmt.Sprintf(`#(%s=="%s")#`, key, value)
		res := gjson.Parse(file).Get(path)
		return res.String()
	} else {
		return file
	}
}

// gets rune from some common csv delimiters
func getComma(comma string) rune {
	switch comma {
	case ",":
		return ','
	case "|":
		return '|'
	case ";":
		return ';'
	case "~":
		return '~'
	default:
		return ','
	}
}
