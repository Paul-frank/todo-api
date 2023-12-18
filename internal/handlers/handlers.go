package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
	default:
        w.Header().Set("Content-Type", "application/json") // Senden einer Fehlermeldung, wenn eine nicht unterstützte Methode verwendet wird
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(struct {
            Error string `json:"error"`
        }{
            Error: "Nicht unterstützte Methode",
        })
    }
}

func ToDoParameterHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method{
	case http.MethodGet:
		GetToDoById(w, r)	// GET /todo/{id}: Abrufen eines spezifischen ToDo-Eintrags.
	case http.MethodPatch:
		PatchToDoById(w, r)	// PATCH /todo/{id}: Aktualisieren eines ToDo-Eintrags.
	default:
        w.Header().Set("Content-Type", "application/json") // Senden einer Fehlermeldung, wenn eine nicht unterstützte Methode verwendet wird
        w.WriteHeader(http.StatusMethodNotAllowed)
        json.NewEncoder(w).Encode(struct {
            Error string `json:"error"`
        }{
            Error: "Nicht unterstützte Methode",
        })
    }
}


func PatchToDoById(w http.ResponseWriter, r *http.Request){
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Umwandeln in neue Todo Instanz
	var updatedToDo models.ToDo
	err = json.NewDecoder(r.Body).Decode(&updatedToDo)
	if err != nil{
		http.Error(w, "Request Body konnte nicht decoded werden", http.StatusBadRequest)
	}

	// Bilden des SQL Strings
	args := []interface{}{} 		// -> Slice vom Typ Interface um Argumente der unterschiedlichen Typen aufzunehmen
	query := "UPDATE todos SET "	// -> SQL Execution String

	if updatedToDo.Title != ""{
		query += "title = ?, "
		args = append(args, updatedToDo.Title)
	}
	if updatedToDo.Description != ""{
		query += "description = ?, "
		args = append(args, updatedToDo.Description)
	}
	if updatedToDo.Category != ""{
		query += "category = ?, "
		args = append(args, updatedToDo.Category)
	}
	if updatedToDo.Order != 0{ //! Sonderfall -> die Reihenfolge der anderen Todos muss agepasst werde
		query += "`order` = ?, "
		args = append(args, updatedToDo.Order)
		err := updateOrder(todoID, updatedToDo.Order)
		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
            return
		}
	}
	if updatedToDo.Completed {
    	query += "completed = ?, "
    	args = append(args, updatedToDo.Completed)
	}

	if len(args) > 0{
		query += "updated_at = ?"
		args = append(args, time.Now())

	} else {
		http.Error(w, "Keine gültigen Parameter im Request Body", http.StatusBadRequest)
		return
	}

	query += " WHERE id = ?"
	args = append(args, todoID)

	_, err = database.Connection.Exec(query, args...)
	if err != nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)		// senden von Statuscode an den Client
	json.NewEncoder(w).Encode(struct {  // senden von einer Bestätigung an den Client	
        Message string `json:"message"`
    }{
        Message: "ToDo erfolgreich aktualisiert",
    })
}

func GetToDoById(w http.ResponseWriter, r *http.Request){
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var todo models.ToDo
	err = database.Connection.QueryRow("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed FROM todos WHERE id = ?", todoID).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Category, &todo.Order, &todo.CreatedAt, &todo.UpdatedAt, &todo.Completed)
	if err != nil{
		if err == sql.ErrNoRows{
			http.Error(w, "ToDo nicht gefunden", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)		// senden von Statuscode an den Client
	json.NewEncoder(w).Encode(todo)  	// senden von der erstellten ToDo an den Client
}

func GetTodosByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Nur Get Methode erlaubt", http.StatusMethodNotAllowed)
		return
	}

	// Parameter UserID auslesen und prüfen
	userID := strings.TrimPrefix(r.URL.Path, "/todo/user/")
	if userID == "" {
		http.Error(w, "UserID fehlt", http.StatusBadRequest)
		return
	}

	// Select per SQL Befehl an Datenbank
	result, err := database.Connection.Query("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed FROM todos WHERE user_id = ?", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func updateOrder(todoID int64, newOrder int) error{
	var currentOrder,userID, maxOrder int

	// Abrufen der aktuellen Position (order) und der UserID
	err := database.Connection.QueryRow("SELECT `order`, user_id FROM todos WHERE id = ?", todoID).Scan(&currentOrder, &userID)
	if err != nil{
		return err
	}

	// Abrufen der maximalen Postion (order)
	err = database.Connection.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", userID).Scan(&maxOrder)
    if err != nil {
        return err
    }

	// Überprüfen ob die neue Position im Bereich der gültigen Werte liegt
	if newOrder > maxOrder {
        return errors.New("Die neue Order ist größer als die maximale erlaubte Order")
    }

	// Anpassen der Position (order) der anderen ToDos
	if newOrder > currentOrder {
		_, err = database.Connection.Exec("UPDATE todos SET `order` = `order` - 1 WHERE `order` > ? AND `order` <= ? AND user_id = ?", currentOrder, newOrder, userID)
	} else if newOrder < currentOrder {
		_, err = database.Connection.Exec("UPDATE todos SET `order` = `order` + 1 WHERE `order` >= ? AND `order` < ? AND user_id = ?", newOrder, currentOrder, userID)
	} else {
		err = errors.New("Die ToDo hat bereits diese Position")
	}
	
	if err != nil{
		return err
	}

	return nil
}