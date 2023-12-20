package handlers

import (
	"database/sql"
	"encoding/json"
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
		sendErrorResponse(w, http.StatusBadRequest, "Nicht unterstützte Methode")
    }
}

func ToDoParameterHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method{
	case http.MethodGet:
		getToDoById(w, r)	// GET /todo/{id}: Abrufen eines spezifischen ToDo-Eintrags.
	case http.MethodPatch:
		patchToDoById(w, r)	// PATCH /todo/{id}: Aktualisieren eines ToDo-Eintrags.
	case http.MethodDelete:
		deleteToDoById(w,r) // DELETE /todo/{id}: löschen eines ToDo-Eintrags.
	default:
		sendErrorResponse(w, http.StatusBadRequest, "Nicht unterstützte Methode")
    }
}

func patchToDoById(w http.ResponseWriter, r *http.Request){ //! Erledigt -> SQL Transaktionen wurden benutzt! + getestet

	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige todo_id")
		return
	}

	// Umwandeln in neue Todo Instanz
	var updatedToDo models.ToDo
	err = json.NewDecoder(r.Body).Decode(&updatedToDo)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Request Body konnte nicht decodiert werden")
		return
	}

	// Beginn der Transaktion
	tx, err := database.Connection.Begin()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Beenden wenn geteilte ToDo
	var currentOriginalID int
	err = tx.QueryRow("SELECT original_todo_id FROM todos WHERE id = ?", todoID).Scan(&currentOriginalID)
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if currentOriginalID != 0{
		sendErrorResponse(w, http.StatusBadRequest, "Geteilte Todo kann nicht angepasst werden")
		return
	}

	// Wenn Position sich verändert, dann ...
	if updatedToDo.Order != 0{
		// Abrufen der aktuellen Position (order) und der UserID
		var currentOrder,userID, maxOrder int
		err = tx.QueryRow("SELECT `order`, user_id FROM todos WHERE id = ?", todoID).Scan(&currentOrder, &userID)
		if err != nil{
			tx.Rollback()
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Abrufen der maximalen Postion (order)
		err = tx.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", userID).Scan(&maxOrder)
    	if err != nil {
			tx.Rollback()
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
        	return
		}
		// Überprüfen ob die neue Position (order) im Bereich der gültigen Werte liegt
		if updatedToDo.Order > maxOrder {
			tx.Rollback()
			sendErrorResponse(w, http.StatusBadRequest, "Die neue Position ist größer als die maximal erlaubte Position")
    	    return
    	}
		if updatedToDo.Order == currentOrder {
			tx.Rollback()
			sendErrorResponse(w, http.StatusBadRequest, "Die neue Position ist die gleiche wie die alte Position")
        	return
		}
		// Anpassen der Position (order) der anderen ToDos
		if updatedToDo.Order > currentOrder {
			_, err = tx.Exec("UPDATE todos SET `order` = `order` - 1 WHERE `order` > ? AND `order` <= ? AND user_id = ?", currentOrder, updatedToDo.Order, userID)
		} else if updatedToDo.Order < currentOrder {
			_, err = tx.Exec("UPDATE todos SET `order` = `order` + 1 WHERE `order` >= ? AND `order` < ? AND user_id = ?", updatedToDo.Order, currentOrder, userID)
		}	
		if err != nil{
			tx.Rollback()
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return 
		}
	}

	// Bilden des SQL Strings für die Aktualisierung
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
	if updatedToDo.Order != 0{
		query += "`order` = ?, "
		args = append(args, updatedToDo.Order)
	}

	if len(args) > 0{
		query += "updated_at = ?"
		args = append(args, time.Now())
	} else {
		sendErrorResponse(w, http.StatusBadRequest, "Keine gültigen Parameter im Request Body")
		return
	}

	query += " WHERE id = ?"
	args = append(args, todoID)

	// Ausführen des SQL Strings für die Aktualisierung der ausgewählten ToDo
	_, err = tx.Exec(query, args...)
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Commit der Transaktion
	err = tx.Commit()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)		
	json.NewEncoder(w).Encode(struct {	
        Message string `json:"message"`
    }{
        Message: "ToDo erfolgreich aktualisiert",
    })
}

func getToDoById(w http.ResponseWriter, r *http.Request){ //! Erledigt + getestet
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// SQL Select Abfrage zum einlesen und Umwandeln in eine ToDo Instanz 
	var todo models.ToDo
	err = database.Connection.QueryRow("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id FROM todos WHERE id = ?", todoID).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Category, &todo.Order, &todo.CreatedAt, &todo.UpdatedAt, &todo.Completed, &todo.OriginalID)
	if err != nil{
		if err == sql.ErrNoRows{
			sendErrorResponse(w, http.StatusBadRequest, "Ungültige todo_id")
			return
		}
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(todo)
}

func createTodo(w http.ResponseWriter, r *http.Request) { //! Erledigt + getestet

	var newTodo models.ToDo // erstellen einer neuen ToDo Instanz

	// Überprüfen ob Json in Struct ToDo umgewandelt werden kann
	err := json.NewDecoder(r.Body).Decode(&newTodo)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Überprüfen ob alle Parameter enthalten
	if newTodo.UserID == 0 || newTodo.Title == "" || newTodo.Description == ""{
		missingFields := []string{}
		if newTodo.UserID == 0 {
			missingFields = append(missingFields, "UserID")
		}
		if newTodo.Title == ""{
			missingFields = append(missingFields, "Titel")
		}
		if newTodo.Description == "" {
			missingFields = append(missingFields, "Beschreibung")
		}

		errorMessage := strings.Join(missingFields,", ") + " fehlt"
		sendErrorResponse(w, http.StatusBadRequest, errorMessage)
    	return
	}

	if newTodo.Category == "" {
		newTodo.Category = "no category"
    }

	// Ermitteln der Order für die UserID
	var maxOrderPtr *int
	err = database.Connection.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", newTodo.UserID).Scan(&maxOrderPtr) // Wenn keine Zeile vorhanden ist gibt Max() Null zurück -> Go kann kein Null in int konventieren -> Zwischenschritt mit Pointer
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	var maxOrder int
	if maxOrderPtr != nil{
		maxOrder = *maxOrderPtr
	} else {
		maxOrder = 0 // 0 wenn der User noch keine ToDos hat
	}

	newTodo.Order = maxOrder + 1

	// SQL Befehl zum Einfügen
	result, err := database.Connection.Exec("INSERT INTO todos (user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	newTodo.UserID, newTodo.Title, newTodo.Description, newTodo.Category, newTodo.Order, time.Now(), time.Now(), false, 0) // -> "false", da neue ToDo nicht schon erledigt sein kann
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// newToDo.ID auslesen für API Response (DB vergibt automatisch IDs)
	id, err := result.LastInsertId()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	newTodo.ID = int(id);

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newTodo)
}

func deleteToDoById (w http.ResponseWriter, r *http.Request){ //! Erledigt -> SQL Transaktionen wurden benutzt! + getestet
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige todo_id")
		return
	}

	// Beginn der Transaktion
	tx, err := database.Connection.Begin()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Anpassen der Position (order)
	var currentOrder, userID int

	err = tx.QueryRow("SELECT `order`, user_id FROM todos WHERE id = ?", todoID).Scan(&currentOrder, &userID) // Position (order) und userID auslesen
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusBadRequest, "todo_id nicht vorhanden")
		return
	}

	_, err = tx.Exec("UPDATE todos SET `order` = `order` - 1 WHERE `order` > ? AND user_id = ?", currentOrder, userID) // Positionen (order) der anderen ToDos anpassen 
    if err != nil {
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
        return
    }

	// Löschen der ToDo
	_, err = tx.Exec("DELETE FROM todos WHERE id = ?", todoID)
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return 
	}

	// Commit der Transaktion
	err = tx.Commit()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(struct {
        Message string `json:"message"`
    }{
        Message: "ToDo erfolgreich gelöscht",
    })
}

func GetTodosByUser(w http.ResponseWriter, r *http.Request) { //! Erledigt + getestet
	if r.Method != "GET" {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Nur Get Methode erlaubt")
		return
	}

	// Parameter UserID auslesen und prüfen
	userID := strings.TrimPrefix(r.URL.Path, "/todo/user/")
	if userID == "" {
		sendErrorResponse(w, http.StatusBadRequest, "UserID fehlt")
		return
	}

	// Select per SQL Befehl an Datenbank
	result, err := database.Connection.Query("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id FROM todos WHERE user_id = ?", userID)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer result.Close()

	if !result.Next(){
		sendErrorResponse(w, http.StatusBadRequest, "User mit dieser ID nicht vorhanden")
		return
	}

	todos := []models.ToDo{} // Erstellen eines Slice von todos

	// Jede SQL Zeile in todo umwandeln und an das Slice todos anfügen
	for result.Next(){
		var todo models.ToDo
		err := result.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Category, &todo.Order, &todo.CreatedAt, &todo.UpdatedAt, &todo.Completed, &todo.OriginalID)
		if err != nil{
			sendErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		todos = append(todos, todo)
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)	
	json.NewEncoder(w).Encode(todos) 
}

func ShareToDoByID (w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST"{
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Nur Post Methode erlaubt")
		return
	}

	// Parameter auslesen und prüfen
	pathSegments := strings.Split(r.URL.Path, "/") // teilt den Path in seine Bestandteile

	if len(pathSegments) < 5{
		sendErrorResponse(w, http.StatusBadRequest, "Fehlende Parameter")
		return
	}

	todoID, err := strconv.ParseInt(pathSegments[3], 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige TodoID")
		return
	}

	userID, err := strconv.ParseInt(pathSegments[4], 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige UserID")
		return
	}

	// Prüfen ob todoID vorhanden ist und Todo einlesen
	var todo models.ToDo
	err = database.Connection.QueryRow("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed FROM todos WHERE id = ?", todoID).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Category, &todo.Order, &todo.CreatedAt, &todo.UpdatedAt, &todo.Completed)
	if err != nil{
		if err == sql.ErrNoRows{
			sendErrorResponse(w, http.StatusBadRequest, "Ungültige TodoID")
			return
		}
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Prüfen ob userID vorhanden ist --> ich hab keine User Tabelle deshalb die Daten aus den todos
	var userExists bool
	err = database.Connection.QueryRow("SELECT EXISTS(SELECT 1 FROM todos WHERE user_id = ?)", userID).Scan(&userExists)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !userExists {
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige UserID")
		return
	}

	// Ermitteln der Order für die UserID
	var maxOrderPtr *int
	err = database.Connection.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", todo.UserID).Scan(&maxOrderPtr) // Wenn keine Zeile vorhanden ist gibt Max() Null zurück -> Go kann kein Null in int konventieren -> Zwischenschritt mit Pointer
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	var maxOrder int
	if maxOrderPtr != nil{
		maxOrder = *maxOrderPtr
	} else {
		maxOrder = 0 // 0 wenn der User noch keine ToDos hat
	}

	todo.Order = maxOrder + 1
	todo.Category = "shared"
	todo.OriginalID = int(todoID)

	// SQL Befehl zum Einfügen
	_ , err = database.Connection.Exec("INSERT INTO todos (user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	todo.UserID, todo.Title, todo.Description, todo.Category, todo.Order, time.Now(), time.Now(), todo.Completed, todo.OriginalID) // -> "false", da neue ToDo nicht schon erledigt sein kann
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{
		Message: "ToDo erfolgreich geteilt",
	})
}

func UpdateToDoStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PATCH" {
        sendErrorResponse(w, http.StatusMethodNotAllowed, "Nur PATCH Methode erlaubt")
        return
    }

	pathSegments := strings.Split(r.URL.Path, "/")
    if len(pathSegments) < 4 {
        sendErrorResponse(w, http.StatusBadRequest, "Fehlende Parameter")
        return
    }

    // Die todo_id befindet sich im vierten Segment des Pfades
    todoID, err := strconv.ParseInt(pathSegments[3], 10, 0)
    if err != nil {

        sendErrorResponse(w, http.StatusBadRequest, "Ungültige TodoID")
        return
    }

	// Request Body auslesen
	var updatedToDo models.ToDo
	err = json.NewDecoder(r.Body).Decode(&updatedToDo)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Request Body konnte nicht decodiert werden")
		return
	}

	// Start der Transaktion
	tx, err := database.Connection.Begin()
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	
	// Abrufen der original_todo_id
	var originalTodoID int
	err = tx.QueryRow("SELECT original_todo_id FROM todos WHERE id = ?", todoID).Scan(&originalTodoID)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

    // Update-Logik
    if originalTodoID == 0 { // Aktuelles ToDo ist das Original
        // Aktualisieren des Originals und aller verknüpften ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE id = ? OR original_todo_id = ?", updatedToDo.Completed, todoID, todoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
    } else { // Aktuelles ToDo ist eine Kopie
        // Aktualisieren nur des Original-ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE id = ?", updatedToDo.Completed, originalTodoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
        // Zusätzlich Aktualisieren aller verknüpften ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE original_todo_id = ?", updatedToDo.Completed, originalTodoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
    }

	// Commit der Transaktion
	if err := tx.Commit(); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{
		Message: "ToDo-Status erfolgreich aktualisiert",
	})
}

type ErrorResponse struct{
	Statuscode 	int		`json:"status_code"`
	Error 		string	`json:"error"`
}

func sendErrorResponse (w http.ResponseWriter, statusCode int, errorMessage string){
	w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(ErrorResponse{
		Statuscode: statusCode,
		Error: errorMessage,
	})
}