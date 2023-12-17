package models

import (
	"time"
)

type ToDo struct {
	ID 			int 		`json:"id"`				// ID der ToDo
	UserID 		int 		`json:"user_id"`		// ID des Benutzers der die ToDo erstellt hat	
	Title 		string 		`json:"title"`			// Titel der ToDo
	Description string 		`json:"description"`	// Beschreibung der ToDo
	Category	string		`json:"category"`		// Kategorie
	Order		int			`json:"order"`			// Postion der Todo -> User abhängig
	CreatedAt 	time.Time 	`json:"created_at"`		// Erstellungsdatum
	UpdatedAt 	time.Time 	`json:"updated_at"`		// Datum der letzten Änderung
	Completed 	bool 		`json:"completed"`		// Status ob Todo erledigt
}