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


func patchToDoById(w http.ResponseWriter, r *http.Request){

	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige todo_id")
		return
	}

	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
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
	var userID, currentOriginalID int
	err = tx.QueryRow("SELECT user_id, original_todo_id FROM todos WHERE id = ?", todoID).Scan(&userID, &currentOriginalID)
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if currentOriginalID != 0{
		sendErrorResponse(w, http.StatusBadRequest, "Geteilte Todo kann nicht angepasst werden")
		return
	}

	// Authentifizierung prüfen
    if !authenticateUser(userID, secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
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

func getToDoById(w http.ResponseWriter, r *http.Request){
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
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

	// Authentifizierung prüfen
    if !authenticateUser(todo.UserID, secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
        return
    }

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(todo)
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
        return
    }

	var newTodo models.ToDo // erstellen einer neuen ToDo Instanz

	// Überprüfen ob Json in Struct ToDo umgewandelt werden kann
	err := json.NewDecoder(r.Body).Decode(&newTodo)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}


	// Überprüfen, ob die UserID in der User-Tabelle existiert
	var exists bool
	err = database.Connection.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", newTodo.UserID).Scan(&exists)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !exists {
		sendErrorResponse(w, http.StatusBadRequest, "UserID existiert nicht")
		return
	}

	// Authentifizierung prüfen, sonnst kann ein fremder User für mich eine Todo erstellen
    if !authenticateUser(newTodo.UserID, secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
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
	_ , err = database.Connection.Exec("INSERT INTO todos (user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	newTodo.UserID, newTodo.Title, newTodo.Description, newTodo.Category, newTodo.Order, time.Now(), time.Now(), false, 0) // -> "false", da neue ToDo nicht schon erledigt sein kann
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Senden der Antwort
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
        Message string `json:"message"`
    }{
        Message: "ToDo erfolgreich erstellt",
    })
}

func deleteToDoById (w http.ResponseWriter, r *http.Request){
	// Parameter Id auslesen und prüfen
	todoID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige todo_id")
		return
	}

    // Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
        return
    }

	// Beginn der Transaktion
	tx, err := database.Connection.Begin()
	if err != nil{
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Prüfen ob todoID vorhand und userID auslesen
	var currentOrder, userID int
	err = tx.QueryRow("SELECT `order`, user_id FROM todos WHERE id = ?", todoID).Scan(&currentOrder, &userID) // Position (order) und userID auslesen
	if err != nil{
		tx.Rollback()
		sendErrorResponse(w, http.StatusBadRequest, "todo_id nicht vorhanden")
		return
	}

	// Authentifizierung prüfen
    if !authenticateUser(int(userID), secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
        return
    }

	// Anpassen der Position (order)
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

func GetTodosByUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Nur Get Methode erlaubt")
		return
	}

    // UserID auslesen
	userID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/todo/user/"), 10, 0)
	if err != nil{
		sendErrorResponse(w, http.StatusBadRequest, "Ungültige userID")
		return
	}
	
	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
        return
    }

	// Authentifizierung prüfen
    if !authenticateUser(int(userID), secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
        return
    }

	// Überprüfen, ob die UserID existiert
    var exists bool
    err = database.Connection.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
    if err != nil {
        sendErrorResponse(w, http.StatusInternalServerError, err.Error())
        return
    }
    if !exists {
        sendErrorResponse(w, http.StatusBadRequest, "User mit dieser ID nicht vorhanden")
        return
    }

	// Select per SQL Befehl an Datenbank
	result, err := database.Connection.Query("SELECT id, user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id FROM todos WHERE user_id = ?", userID)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer result.Close()

	// Erstellen eines Slice von todos
	todos := []models.ToDo{} 

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

	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
        return
    }

	// Authentifizierung prüfen für die ursprüngliche ToDo
    var originalUser int
    err = database.Connection.QueryRow("SELECT user_id FROM todos WHERE id = ?", todoID).Scan(&originalUser)
    if err != nil {
        sendErrorResponse(w, http.StatusInternalServerError, err.Error())
        return
    }
    if !authenticateUser(originalUser, secretKey) {
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
        return
    }

	// Prüfen ob todoID vorhanden ist und Todo einlesen
	var originalTodo models.ToDo
	err = database.Connection.QueryRow("SELECT title, description, category, completed FROM todos WHERE id = ?", todoID).Scan(&originalTodo.Title, &originalTodo.Description, &originalTodo.Category, &originalTodo.Completed)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, http.StatusBadRequest, "Ungültige TodoID")
			return
		}
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Prüfen ob userID vorhanden ist 
	var userExists bool
	err = database.Connection.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&userExists)
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
	err = database.Connection.QueryRow("SELECT MAX(`order`) FROM todos WHERE user_id = ?", userID).Scan(&maxOrderPtr) // Wenn keine Zeile vorhanden ist gibt Max() Null zurück -> Go kann kein Null in int konventieren -> Zwischenschritt mit Pointer
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

	// Erstellen der neuen Todo
	var newTodo models.ToDo
	newTodo.UserID = int(userID)
	newTodo.Title = originalTodo.Title
	newTodo.Description = originalTodo.Description
	newTodo.Category = "shared"
	newTodo.Order = maxOrder + 1
	newTodo.Completed = originalTodo.Completed
	newTodo.OriginalID = int(todoID)

	// SQL Befehl zum Einfügen
	_ , err = database.Connection.Exec("INSERT INTO todos (user_id, title, description, category, `order`, created_at, updated_at, completed, original_todo_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	newTodo.UserID, newTodo.Title, newTodo.Description, newTodo.Category, newTodo.Order, time.Now(), time.Now(), newTodo.Completed, newTodo.OriginalID) // -> "false", da neue ToDo nicht schon erledigt sein kann
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

	// Parameter auslesen und prüfen
	pathSegments := strings.Split(r.URL.Path, "/")
    if len(pathSegments) < 4 {
        sendErrorResponse(w, http.StatusBadRequest, "Fehlende TodoID")
        return
    }

    todoID, err := strconv.ParseInt(pathSegments[3], 10, 0)
    if err != nil {
        sendErrorResponse(w, http.StatusBadRequest, "Ungültige TodoID")
        return
    }

	// Secret Key aus dem Header auslesen
    secretKey := r.Header.Get("Secret-Key")
    if secretKey == "" {
        sendErrorResponse(w, http.StatusBadRequest, "Secret Key fehlt")
        return
    }

	// Request Body auslesen
	var updatedTodo models.ToDo
	err = json.NewDecoder(r.Body).Decode(&updatedTodo)
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
	
	// Abrufen der userID und der original_todo_id
	var userID, originalTodoID int
	err = tx.QueryRow("SELECT user_id, original_todo_id FROM todos WHERE id = ?", todoID).Scan(&userID, &originalTodoID)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

    // Authentifizierung prüfen
    if !authenticateUser(userID, secretKey) {
        tx.Rollback()
        sendErrorResponse(w, http.StatusUnauthorized, "Nicht autorisiert")
        return
    }

    // Update-Logik
    if originalTodoID == 0 { // Aktuelles ToDo ist das Original
        // Aktualisieren des Originals und aller verknüpften ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE id = ? OR original_todo_id = ?", updatedTodo.Completed, todoID, todoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
    } else { // Aktuelles ToDo ist eine Kopie
        // Aktualisieren nur des Original-ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE id = ?", updatedTodo.Completed, originalTodoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
        // Zusätzlich Aktualisieren aller verknüpften ToDos
        if _, err = tx.Exec("UPDATE todos SET completed = ? WHERE original_todo_id = ?", updatedTodo.Completed, originalTodoID); err != nil {
            tx.Rollback()
            sendErrorResponse(w, http.StatusInternalServerError, err.Error())
            return
        }
    }
		/*
		Nicht behandelte Edge Cases:
			- Ich kann Todos mit mir selbst teilen
			- Wenn Todo auf eine gelöschte Origanl Todo verweist -> was dann? -> kein Fehler -> der Status der geteilten Todo verändert sich 
			- Wenn eine Todo geteilt wird und die geteilte Todo wieder geteilt wird -> was dann? -> Nur eine Anpassung der angesprochenen Todo und derer original Todo, die urpsüngliche Todo bleibt unverändert bis zu den Zeitpunkt wo Sie oder die geteilte Version angesprochen werden
		*/

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

func authenticateUser(userID int, secretKey string) bool {

    // Logik zum Überprüfen der Authentifizierung
    var storedSecretKey string
    err := database.Connection.QueryRow("SELECT secret_key FROM users WHERE id = ?", userID).Scan(&storedSecretKey)
    if err != nil {
        return false
    }

    return secretKey == storedSecretKey
}