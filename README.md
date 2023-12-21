# ToDo API Check24 - Beschreibung

Die ToDo API bietet eine effiziente Lösung zum Verwalten von ToDo-Einträgen in einer SQLite-Datenbank. Die Hauptfunktionen der API werden im folgenden vorgestellt.

Um zu gewährleisten, dass Benutzer nur auf ihre eigenen ToDos zugreifen können, nutzt die API ein Authentifizierungsschema basierend auf einem "Secret-Key", der im Request-Header übermittelt wird. Dieser Ansatz wurde gewählt, da ich momentan noch mehr Erfahrung im Umgang mit komplexeren Authentifizierungsschemata wie OAuth sammle.

## Endpunkte

### /todo
> POST - Erstellt einen neuen ToDo-Eintrag in der Datenbank
```json
Body:
{
"user_id": 2,
"title": "User2 Test",
"description": "Das ist der allerletzte Test",
"category": "final_test2"
}
```


### /todo/{todoID}
> GET - Ruft einen spezifischen ToDo-Eintrag anhand seiner ID ab

> DELETE - Löscht einen spezifischen ToDo-Eintrag

> PATCH - Aktualisiert einen spezifischen ToDo-Eintrag (Titel, Beschreibung, Kategorie, Reihenfolge)
```json
Body:
{
"title": "Test",
"description": "Ich bin ein Test",
"category": "tests"
}
```

### /todo/user/{userID}
> GET - Ruft alle ToDo-Einträge eines spezifischen Benutzers ab

### /todo/share/{todoID}/{userID}
> POST - Teilt einen spezifischen ToDo-Eintrag mit einem anderen Benutzer

### /todo/status/{todoID}
> PATCH - Aktualisiert den Status (erledigt/nicht erledigt) eines ToDo-Eintrags und aller verknüpften geteilten ToDos.
```json
Body:
{
"completed": true
}
```
