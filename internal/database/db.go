package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct { // Datenbankverbindung
    Connection *sql.DB // Zeiger auf sql.DB-Instanz, die die Verbindung zur Datenbank enthält
}

func NewDatabase(dataSourceName string) *Database {
    db, err := sql.Open("sqlite3", dataSourceName) // Öffnen der Datenbankverbindung
    if err != nil {
        log.Fatal(err)
    }

    return &Database{	// Rückgabe einer Database Instanz
        Connection: db, // Zeiger auf offene Datenbankverbindung setzen
    }
}

// Methode zum schließen der Datanbankverbindung
func (db *Database) Close() {
    db.Connection.Close()
}