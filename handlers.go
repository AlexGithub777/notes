package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/icza/session"
)

type noteData struct {
	Username string
	Notes    []Note
}


func (a *App) listHandler(w http.ResponseWriter, r *http.Request) {
    a.isAuthenticated(w, r)

    sess := session.Get(r)
    username := "[guest]"

    if sess != nil {
        username = sess.CAttr("username").(string)
		a.username = username
		
    }

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Retrieve all notes
    notes, err := a.retrieveNotes(username)
    if err != nil {
        checkInternalServerError(err, w)
        return
    }

    // Retrieve all shared notes with privileges
    sharedNotes, err := a.retrieveSharedNotesWithPrivileges(username)
    if err != nil {
        checkInternalServerError(err, w)
        return
    }

    // Get the list of all users
    allUsers, err := a.getAllUsers(username)
    if err != nil {
        // Handle the error appropriately (e.g., log it or show an error page)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    var funcMap = template.FuncMap{
        "addOne": func(i int) int {
            return i + 1
        },
    }

    // Pass the shared notes with privileges to the template
    data := struct {
        Username      string
        Notes         []Note
        AllUsers      []User
        SharedNotes   []Note
    }{
        Username:      username,
        Notes:         notes,
        AllUsers:      allUsers,
        SharedNotes:   sharedNotes,
    }

    t, err := template.New("list.html").Funcs(funcMap).ParseFiles("tmpl/list.html")
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    var buf bytes.Buffer
    err = t.Execute(&buf, data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=UTF-8")
    buf.WriteTo(w)
}

// Add the searchNotesHandler function
func (a *App) searchNotesHandler(w http.ResponseWriter, r *http.Request) {
    searchQuery := r.FormValue("searchQuery")
    fmt.Printf("%s", searchQuery)
	

    // Query your database using FTS to search for notes based on searchQuery
    // Replace this with your actual database query
    results, err := a.searchNotesInDatabase(searchQuery, a.username)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Print the search results to the console
    for i, result := range results {
        fmt.Printf("Result %d:\n", i+1)
        fmt.Printf("ID: %d\n", result.ID)
        fmt.Printf("Title: %s\n", result.Title)
        fmt.Printf("NoteType: %s\n", result.NoteType)
        fmt.Printf("Description: %s\n", result.Description)
        fmt.Printf("NoteCreated: %s\n", result.NoteCreated)
        fmt.Printf("TaskCompletionTime: %s (Valid: %t)\n", result.TaskCompletionTime.String, result.TaskCompletionTime.Valid)
        fmt.Printf("TaskCompletionDate: %s (Valid: %t)\n", result.TaskCompletionDate.String, result.TaskCompletionDate.Valid)
        fmt.Printf("NoteStatus: %s (Valid: %t)\n", result.NoteStatus.String, result.NoteStatus.Valid)
        fmt.Printf("NoteDelegation: %s (Valid: %t)\n", result.NoteDelegation.String, result.NoteDelegation.Valid)
    }
    
    // Pass the shared notes with privileges to the template
    searchData := struct {
        SearchResults   []Note
    }{
        SearchResults:  results,
    }


    // Render the search results in your template
    t, err := template.New("search_results.html").ParseFiles("tmpl/search_results.html")
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Pass the search results to the template
    if err := t.Execute(w, searchData); err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
}

// Implement the searchNotesInDatabase function to query your database using FTS
func (a *App) searchNotesInDatabase(searchQuery string, username string) ([]Note, error) {
    // Implement your database query using FTS to search for notes based on searchQuery
    // Replace this with your actual database query logic

    
    
	fmt.Print(username)
    rows, err := a.db.Query("SELECT id, title, notetype, description, notecreated, taskcompletiondate, taskcompletiontime, notestatus, notedelegation FROM notes WHERE fts_text @@ to_tsquery('english', $1) AND owner = $2", searchQuery, username)
    if err != nil {
        return nil, err
    }
	
    defer rows.Close()
	var notes []Note
    for rows.Next() {
        var id int
        var title, noteType, description, taskCompletionDate, taskCompletionTime, noteStatus, noteDelegation string
		var noteCreated time.Time
        // Populate the note struct from the database result
        if err := rows.Scan(&id, &title, &noteType, &description, &noteCreated, &taskCompletionDate, &taskCompletionTime, &noteStatus, &noteDelegation); err != nil {
            fmt.Println("error")
			return nil, err
        }
		
		var note Note
		note.ID = id
		note.Title = title
		note.Description = description
		note.NoteType = noteType
		note.Description = description
		note.NoteCreated = noteCreated
		note.TaskCompletionDate.String = taskCompletionDate
		note.TaskCompletionTime.String = taskCompletionTime
		note.NoteStatus.String = noteStatus
		note.NoteDelegation.String = noteDelegation

		notes = append(notes, note)
        
        // Print the result to the console
        fmt.Printf("Result - ID: %d, Title: %s, Description: %s, ... (add other fields)\n", id, title, description)
    }



    return notes, nil
}


func (a *App) retrieveNotes(username string) ([]Note, error) {
	
    rows, err := a.db.Query("SELECT * FROM notes WHERE owner = $1 ORDER BY id", username)
    if err != nil {
        return nil, err
    }

    var notes []Note
    for rows.Next() {
        var note Note
        err := rows.Scan(
            &note.ID,
            &note.Title,
            &note.NoteType,
            &note.Description,
            &note.NoteCreated,
            &note.TaskCompletionTime,
            &note.TaskCompletionDate,
            &note.NoteStatus,
            &note.NoteDelegation,
            &note.Owner,
            &note.FTSText,
        )
        if err != nil {
            return nil, err
        }
        notes = append(notes, note)
    }

    return notes, nil
}

func (a *App) retrieveSharedNotesWithPrivileges(username string) ([]Note, error) {
    rows, err := a.db.Query(`
        SELECT n.*, us.privileges
        FROM notes n
        INNER JOIN user_shares us ON n.id = us.note_id
        WHERE us.username = $1
        ORDER BY n.id
    `, username)
    if err != nil {
        return nil, err
    }

    var sharedNotes []Note
    for rows.Next() {
        var sharedNote Note
		
        
        err := rows.Scan(
            &sharedNote.ID,
            &sharedNote.Title,
            &sharedNote.NoteType,
            &sharedNote.Description,
            &sharedNote.NoteCreated,
            &sharedNote.TaskCompletionTime,
            &sharedNote.TaskCompletionDate,
            &sharedNote.NoteStatus,
            &sharedNote.NoteDelegation,
            &sharedNote.Owner,
            &sharedNote.FTSText,
            &sharedNote.Privileges, // Retrieve the 'privileges' field
        )
        if err != nil {
            return nil, err
        }
        
        // Set the 'Privileges' field in the Note struct
        

        sharedNotes = append(sharedNotes, sharedNote)
    }

    return sharedNotes, nil
}




func (a *App) getAllUsers(ownerUsername string) ([]User, error) {
    var users []User

    rows, err := a.db.Query("SELECT username FROM users WHERE username != $1", ownerUsername)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var user User
        if err := rows.Scan(&user.Username); err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return users, nil
}




func (a *App) createHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)

	sess := session.Get(r)
	username := sess.CAttr("username").(string)

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}


    var note Note
	note.Title = r.FormValue("Title")
	note.NoteType = r.FormValue("NoteType")
	note.Description = r.FormValue("Description")
	note.Owner = username // Set the owner ID to the logged-in user's ID (adjust as needed) !!! set to userID
    note.TaskCompletionDate.String = r.FormValue("TaskCompletionDate")
    note.TaskCompletionTime.String = r.FormValue("TaskCompletionTime")
    note.NoteStatus.String = r.FormValue("NoteStatus")
    note.NoteDelegation.String = r.FormValue("NoteDelegation")


	// Save to database
	_, err := a.db.Exec(`
		INSERT INTO notes (title, noteType, description, TaskCompletionDate, TaskCompletionTime, NoteStatus, NoteDelegation, owner, fts_text)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, to_tsvector('english', $1 || ' ' || $2 || ' ' || $3 || ' ' || $4 || ' ' || $5 || ' ' || $6 || ' ' || $7 || ' ' || $8))
	`, note.Title, note.NoteType, note.Description, note.TaskCompletionDate.String, note.TaskCompletionTime.String, note.NoteStatus.String, note.NoteDelegation.String, note.Owner)
	checkInternalServerError(err, w)

	

	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func (a *App) updateHandler(w http.ResponseWriter, r *http.Request) {
    a.isAuthenticated(w, r)

    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    var note Note
    note.ID, _ = strconv.Atoi(r.FormValue("Id")) // Given ID
    note.Title = r.FormValue("Title") 
    note.NoteType = r.FormValue("NoteType")
    note.Description = r.FormValue("Description")
	note.TaskCompletionTime.String = r.FormValue("TaskCompletionTime")
    note.TaskCompletionDate.String = r.FormValue("TaskCompletionDate")
    note.NoteStatus.String = r.FormValue("NoteStatus")
    note.NoteDelegation.String = r.FormValue("NoteDelegation")

    // Update the database
    _, err := a.db.Exec(`
        UPDATE notes SET title=$1, noteType=$2, description=$3,
        taskcompletiontime=$4, taskcompletiondate=$5, notestatus=$6, notedelegation=$7
        WHERE id=$8
    `, note.Title, note.NoteType, note.Description, note.TaskCompletionTime.String,
    note.TaskCompletionDate.String, note.NoteStatus.String, note.NoteDelegation.String, note.ID)
    if err != nil {
        checkInternalServerError(err, w)
        return
    }

    // Redirect back to the list page or another appropriate page
    http.Redirect(w, r, "/list", http.StatusSeeOther)
}


func (a *App) deleteHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	noteID, _ := strconv.Atoi(r.FormValue("Id"))
	// Delete from the database
	_, err := a.db.Exec("DELETE FROM notes WHERE id=$1", noteID)
	checkInternalServerError(err, w)

	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func (a *App) shareHandler(w http.ResponseWriter, r *http.Request) {
    a.isAuthenticated(w, r)

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Extract the shared user's username and privileges from the form
    sharedUsername := r.FormValue("SharedUsername")
    privileges := r.FormValue("Privileges")
    noteID := r.FormValue("Id")
	
	

    // Check if the shared user exists in the users table by username
    var sharedUserID string // Change the type to string
    err := a.db.QueryRow("SELECT username FROM users WHERE username = $1", sharedUsername).Scan(&sharedUserID)
    if err != nil {
        // Handle the case where the shared user does not exist
        // You can display an error message or redirect as needed
        http.Error(w, "Invalid shared user", http.StatusBadRequest)
        return
    }

    // Check if the note with the given ID exists
    var noteExists bool
    err = a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM notes WHERE id = $1)", noteID).Scan(&noteExists)
	fmt.Printf("%t\n", noteExists)
    if err != nil {
        checkInternalServerError(err, w)
        return
    }

    if !noteExists {
        http.Error(w, "Note does not exist", http.StatusBadRequest)
        return
    }

    // Check if there is already an existing entry for the given note and shared user
    var existingShareID int
    err = a.db.QueryRow("SELECT note_id FROM user_shares WHERE note_id = $1 AND username = $2", noteID, sharedUsername).Scan(&existingShareID)
    if err == nil {
        // An existing entry was found, which means the note is already shared with the user
        // You can handle this case by displaying an error message to the user
        http.Error(w, "Note is already shared with this user", http.StatusBadRequest)
        return
    } else if err != sql.ErrNoRows {
        // Handle any other errors that may have occurred during the query
        checkInternalServerError(err, w)
        return
    }

    // If no existing entry was found, proceed with sharing the note
    _, err = a.db.Exec(`
        INSERT INTO user_shares (note_id, username, privileges)
        VALUES ($1, $2, $3)
    `, noteID, sharedUsername, privileges)
    if err != nil {
        checkInternalServerError(err, w)
        return
    }

    // Provide feedback to the user (e.g., "Note shared successfully")

    // Redirect to an appropriate page
    http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func (a *App) removeSharedNoteHandler(w http.ResponseWriter, r *http.Request) {
    a.isAuthenticated(w, r)

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Parse the note ID from the request
    noteID := r.FormValue("noteID")

    // Get the username of the user who wants to remove the shared note
    sess := session.Get(r)
    username := "[guest]"

    if sess != nil {
        username = sess.CAttr("username").(string)
    }

    // Implement the logic to remove the shared note from the user_shares table
    err := a.removeSharedNoteFromUser(username, noteID)
    if err != nil {
        // Handle the error appropriately (e.g., log it or show an error page)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Redirect the user to a success page or back to the list of shared notes
    http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func (a *App) removeSharedNoteFromUser(username string, noteID string) error {
    _, err := a.db.Exec(`
        DELETE FROM user_shares
        WHERE username = $1 AND note_id = $2
    `, username, noteID)
    if err != nil {
        return err
    }

    return nil
}

/*func (a *App) getFilteredUsers(ownerUsername string, noteID int) ([]User, error) {
	fmt.Print("hi")
    var users []User
	fmt.Printf("%d", noteID)

    // Use a subquery to select users who are not in the user_shares table for the given noteID
    query := `
        SELECT username FROM users
        WHERE username != $1
        AND username NOT IN (
            SELECT username FROM user_shares WHERE note_id = $2
        )
    `

    rows, err := a.db.Query(query, ownerUsername, noteID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var user User
        if err := rows.Scan(&user.Username); err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }

    return users, nil
}




func (a *App) shareNoteHandler(w http.ResponseWriter, r *http.Request) {
    noteID := r.FormValue("Id") // Get the noteID from the form data
	noteIDInt, err := strconv.Atoi(noteID)


	sess := session.Get(r)
    username := "[guest]"

    if sess != nil {
        username = sess.CAttr("username").(string)
    }

    // Query your database to get the list of non-shared users for the specified note.
    nonSharedUsers, err := a.getFilteredUsers(username, noteIDInt)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
	fmt.Printf("%v", nonSharedUsers)

    // Redirect the user to a success page or back to the list of shared notes
    http.Redirect(w, r, "/list", http.StatusSeeOther)
}
*/









func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

