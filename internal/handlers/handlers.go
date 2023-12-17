package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	db "github.com/Paul-frank/todo-api/internal/database"
	"github.com/Paul-frank/todo-api/internal/models"
)

var database *db.Database // globaler DB-Pointer

func SetDatabase(db *db.Database) {  // Datenbankverbindung aus der Main übergeben
    database = db
}


func ToDoHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method{
	case http.MethodPost:
		createTodo(w, r) 	// POST /todo: Erstellen eines neuen ToDo-Eintrags
	}
}




func GetToDoById(w http.ResponseWriter, r *http.Request){
	if r.Method != "GET" {
		http.Error(w, "Nur Get Methode erlaubt", http.StatusMethodNotAllowed)
		return
	}

}



func GetTodosByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Nur Get Methode erlaubt", http.StatusMethodNotAllowed)
		return
	}

	// Parameter UserID auslesen und prüfen
	userID := strings.TrimPrefix(r.URL.Path, "/todos/user/")
	if userID == "" {
		http.Error(w, "UserID fehlt", http.StatusBadRequest)
		return
	}

	// Select per SQL Befehl an Datenbank
	result, err := database.Connection.Query("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed FROM todos WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer result.Close()

	todos := []models.ToDo{} // Erstellen eines Slice von todos

	// Jede SQL Zeile in todo umwandeln und an das Slice todos anfügen
	for result.Next(){
		var todo models.ToDo
		err := result.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Category, &todo.Order, &todo.CreatedAt, &todo.UpdatedAt, &todo.Completed)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		todos = append(todos, todo)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)		// senden von Statuscode an den Client
	json.NewEncoder(w).Encode(todos)  	// senden von der erstellten ToDo an den Client
}













func createTodo(w http.ResponseWriter, r *http.Request) {

	var newToDo models.ToDo // erstellen einer neuen ToDo Instanz

	// Überprüfen ob Json in Struct ToDo umgewandelt werden kann
	err := json.NewDecoder(r.Body).Decode(&newToDo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Überprüfen ob alle Parameter enthalten
	if newToDo.UserID == 0 {
		http.Error(w, "UserID ist erforderlich, bitte logge dich ein", http.StatusBadRequest)
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

	if newToDo.Category == "" {
		newToDo.Category = "no category"
    }

	// Ermitteln der Order für die UserID
	var maxOrderPtr *int
	err = database.Connection.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", newToDo.UserID).Scan(&maxOrderPtr) // Wenn keine Zeile vorhanden ist gibt Max() Null zurück -> Go kann kein Null in int konventieren -> Zwischenschritt mit Pointer
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var maxOrder int
	if maxOrderPtr != nil{
		maxOrder = *maxOrderPtr
	} else {
		maxOrder = 0 // 0 wenn der User noch keine ToDos hat
	}

	newToDo.Order = maxOrder + 1

	// Einfügen per SQL Befehl
	result, err := database.Connection.Exec("INSERT INTO todos (user_id, title, description, category, `order`, created_at, updated_at, completed) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
	newToDo.UserID, newToDo.Title, newToDo.Description, newToDo.Category, newToDo.Order, time.Now(), time.Now(), false) // -> "false", da neue ToDo nicht schon erledigt sein kann
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)	// senden von Statuscode an den Client
	json.NewEncoder(w).Encode(newToDo)  // senden von der erstellten ToDo an den Client
}