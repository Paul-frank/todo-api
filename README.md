Die ToDo API ermöglicht das Verwalten von ToDo-Einträgen in einer SQLite-Datenbank. Die Hauptfunktionalitäten umfassen das Erstellen, Aktualisieren, Löschen und Abrufen von ToDo-Einträgen, sowie das Teilen und Aktualisieren des Status von ToDos.

Um zu gewährleisten das ein Benutzer nur seine ToDo sieht und nur selbst ToDos erstellen hab ich mich dazu entschieden eine Authentifizierungsschemas zu implemetieren, das ein "Secret-Key" aus dem Header rausliest, da ich noch nicht zu vile erfahrung mit anderen Authenfizierungschemas wie oAuth habe.

Endpunkte

/todo
POST - Erstellt einen neuen ToDo-Eintrag in der Datenbank
http://localhost:8080/todo
Body:
{
"user_id": 2,
"title": "User2 Test",
"description": "Das ist der allerletzte Test",
"category": "final_test2"
}

/todo/{todoID}
GET - Ruft einen spezifischen ToDo-Eintrag anhand seiner ID ab
http://localhost:8080/todo/1
PATCH - Aktualisiert einen spezifischen ToDo-Eintrag (Titel, Beschreibung, Kategorie, Reihenfolge)
http://localhost:8080/todo/1
Body:
{
"title": "Test",
"description": "Ich bin ein Test",
"category": "tests"
}
DELETE - Löscht einen spezifischen ToDo-Eintrag

/todo/user/{userID}
GET - Ruft alle ToDo-Einträge eines spezifischen Benutzers ab

/todo/share/{todoID}/{userID}
POST - Teilt einen spezifischen ToDo-Eintrag mit einem anderen Benutzer

/todo/status/{todoID}
PATCH - Aktualisiert den Status (erledigt/nicht erledigt) eines ToDo-Eintrags und aller verknüpften geteilten ToDos.
http://localhost:8080/todo/status/1
Body:
{
"completed": true
}
