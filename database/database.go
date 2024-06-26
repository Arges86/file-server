package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const create string = `
CREATE TABLE IF NOT EXISTS "files" (
	"id"	INTEGER NOT NULL,
	"time"	DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"data"	JSON NOT NULL,
	"name"	TEXT NOT NULL DEFAULT '' UNIQUE,
	PRIMARY KEY("id" AUTOINCREMENT)
);`

type Database struct {
	db *sql.DB
}

type File struct {
	ID     int       `json:"id,omitempty"`
	Time   time.Time `json:"last_updated,omitempty"`
	Name   string    `json:"name,omitempty"`
	Data   string    `json:"data,omitempty"`
	Size   string    `json:"size,omitempty"`
	length int64
}

func New(file string) (*Database, error) {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(create); err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

/** lists all files and their timestamps **/
func (c *Database) GetAll() ([]File, error) {
	var output []File
	rows, err := c.db.Query("SELECT id, time, name, length(data) as length FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		i := File{}
		err = rows.Scan(&i.ID, &i.Time, &i.Name, &i.length)
		if err != nil {
			return nil, err
		}
		output = append(output, i)
	}

	for i := range output {
		output[i].Size = byteCountSI(output[i].length)
	}
	return output, nil
}

/** Gets a specific file **/
func (c *Database) GetFile(name string) (string, error) {
	row := c.db.QueryRow("SELECT data FROM files where name =  ?", name)
	file := File{}

	if err := row.Scan(&file.Data); err == sql.ErrNoRows {
		return "", err
	}
	return file.Data, nil
}

/** INSERTS or UPDATES a file based on the `name` **/
func (c *Database) SaveFile(data string, name string) (int, error) {
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

// takes length in bytes and returns SI name of unit
func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
