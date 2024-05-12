package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const file string = "./files.db"

const create string = `
CREATE TABLE IF NOT EXISTS "files" (
	"id"	INTEGER NOT NULL,
	"time"	DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"data"	JSON NOT NULL,
	"name"	TEXT NOT NULL DEFAULT '' UNIQUE,
	PRIMARY KEY("id" AUTOINCREMENT)
);`

type Files struct {
	// mu    sync.Mutex
	// files []File
	db *sql.DB
}

type File struct {
	ID   int       `json:"id,omitempty"`
	Time time.Time `json:"last_updated,omitempty"`
	Name string    `json:"name,omitempty"`
	Data string    `json:"data,omitempty"`
}

func New() (*Files, error) {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(create); err != nil {
		return nil, err
	}

	return &Files{
		db: db,
	}, nil
}

/** lists all files and their timestamps **/
func (c *Files) GetAll() ([]File, error) {
	var output []File
	rows, err := c.db.Query("SELECT id, time, name FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		i := File{}
		err = rows.Scan(&i.ID, &i.Time, &i.Name)
		if err != nil {
			return nil, err
		}
		output = append(output, i)
	}
	return output, nil
}

/** Gets a specific file **/
func (c *Files) GetFile(name string) (string, error) {
	row := c.db.QueryRow("SELECT data FROM files where name =  ?", name)
	file := File{}

	if err := row.Scan(&file.Data); err == sql.ErrNoRows {
		return "", err
	}
	return file.Data, nil
}

/** INSERTS or UPDATES a file based on the `name` **/
func (c *Files) SaveFile(data string, name string) (int, error) {
	res, err := c.db.Exec(
		`INSERT INTO files (data, name)
		VALUES(?,?)
		ON CONFLICT(name) DO UPDATE SET data = ?, time = CURRENT_TIMESTAMP;`, data, name, data)
	if err != nil {
		return 0, err
	}
	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return 0, err
	}
	return int(id), nil
}
