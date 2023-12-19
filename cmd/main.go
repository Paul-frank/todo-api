package main

import (
	"log"
	"net/http"

	"github.com/Paul-frank/todo-api/internal/database"
	"github.com/Paul-frank/todo-api/internal/handlers"
)

func main(){

    db := database.NewDatabase("../internal/database/todo_app_db.db") // Erstelle eine neue Datenbankinstanz
    defer db.Close() // Beenden der Datenbankinstanz

    handlers.SetDatabase(db) // Setze die Datenbankinstanz in den Handlers
    

	http.HandleFunc("/todo", handlers.ToDoHandler)
	http.HandleFunc("/todo/", handlers.ToDoParameterHandler)    //* GET /todos/{userID}: Abrufen eines spezifischen ToDo-Eintrags. 			
	http.HandleFunc("/todo/user/", handlers.GetTodosByUser) 	//* GET /todo/user/{ID}: Abrufen aller ToDo-Einträge des angemeldeten Benutzers


    log.Println("Server startet auf :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

/*
ToDo:
	- Request http://localhost:8080/todos/user/1/100... gibt Statuscode 200 -> soll nicht sein
	// - Exceptions sind nicht als Json formatiert!
	//- Datenbanktransaktion (alle Operationen erfolgreich abgeschlossen sonst Rollback)
	// - Testen!!!

	//GET /todo/user/{ID}: Abrufen aller ToDo-Einträge des angemeldeten Benutzers.
	//POST /todo: Erstellen eines neuen ToDo-Eintrags.
	//GET /todo/{id}: Abrufen eines spezifischen ToDo-Eintrags.
	//PATCH /todo/{id}: Aktualisieren eines ToDo-Eintrags.
	//DELETE /todo/{id}: Löschen eines ToDo-Eintrags.
	//PATCH /todos/{id}/complete: Markieren eines ToDo-Eintrags als erledigt.
	POST /todo/{id}/share: Teilen eines ToDo-Eintrags mit einem anderen Benutzer -> Kategorie shared
*/