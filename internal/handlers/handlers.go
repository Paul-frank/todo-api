package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"

	db "github.com/Paul-frank/todo-api/internal/database"
	"github.com/Paul-frank/todo-api/internal/models"
)

var database *db.Database // globaler DB-Pointer

func SetDatabase(db *db.Database) {  // Datenbankverbindung aus der Main übergeben
    database = db
}


//* POST /createTodo: Erstellen eines neuen ToDo-Eintrags
func CreateToDoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { // Überprüfen ob POST Request
		http.Error(w, "Nur POST Methode erlaubt", http.StatusMethodNotAllowed)
		return
	}

	var newToDo models.ToDo // erstellen einer neuen ToDo Instanz

	// Überprüfen ob Json in Struct ToDo umgewandelt werden kann
	err := json.NewDecoder(r.Body).Decode(&newToDo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Überprüfen ob alle Parameter enthalten
	if newToDo.UserID == 0 {
		http.Error(w, "UserID ist erforderlich, bitte logge dich ein!", http.StatusBadRequest)
        return
    }

	if newToDo.Title == "" {
		http.Error(w, "Titel ist erforderlich", http.StatusBadRequest)
        return
    }

	if newToDo.Description == "" {
		http.Error(w, "Beschreibung ist erforderlich", http.StatusBadRequest)
        return
    }

	// Einfügen per SQL Befehl
	result, err := database.Connection.Exec("INSERT INTO todos (user_id, title, description, created_at, updated_at, completed) VALUES (?, ?, ?, ?, ?, ?)",
	newToDo.UserID, newToDo.Title, newToDo.Description, time.Now(), time.Now(), false) // -> "false", da neue ToDo nicht schon erledigt sein kann
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// newToDo.ID auslesen für API Response (DB vergibt automatisch IDs)
	id, err := result.LastInsertId()
	if err != nil{
		http.Error(w,err.Error(), http.StatusBadRequest)
		return
	}

	newToDo.ID = int(id);

	w.WriteHeader(http.StatusCreated)	// senden von Statuscode an den Client
	json.NewEncoder(w).Encode(newToDo)  // senden von der erstellten ToDo an den Client
}