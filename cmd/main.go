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

    http.HandleFunc("/createTodo", handlers.CreateToDoHandler) //* POST /createTodo: Erstellen eines neuen ToDo-Eintrags

    log.Println("Server startet auf :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

/*
ToDo:
	POST /todos: Erstellen eines neuen ToDo-Eintrags.

	GET /todos: Abrufen aller ToDo-Einträge des angemeldeten Benutzers.
	GET /todos/{id}: Abrufen eines spezifischen ToDo-Eintrags.
	PUT /todos/{id}: Aktualisieren eines ToDo-Eintrags.
	DELETE /todos/{id}: Löschen eines ToDo-Eintrags.
	PATCH /todos/{id}/complete: Markieren eines ToDo-Eintrags als erledigt.

	POST /todos/{id}/share: Teilen eines ToDo-Eintrags mit einem anderen Benutzer.
	GET /todos/shared: Abrufen aller geteilten ToDo-Einträge.
*/