package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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
    }

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Retrieve all notes
    notes, err := a.retrieveNotes(username)
    if err != nil {
        a.checkInternalServerError(err, w)
        return
    }

    // Get the list of all users
    allUsers, err := a.getAllUsers()
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

    data := struct {
        Username string
        Notes    []Note
        AllUsers []User // Add the AllUsers field to the data struct
    }{
        Username: username,
        Notes:    notes,
        AllUsers: allUsers, // Pass the allUsers slice to the template
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

// function to get all users
func (a *App) getAllUsers() ([]User, error) {
    var users []User

    rows, err := a.db.Query("SELECT username FROM users")
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
		INSERT INTO notes (title, noteType, description, TaskCompletionDate, TaskCompletionTime, NoteStatus, NoteDelegation, owner)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	`, note.Title, note.NoteType, note.Description, note.TaskCompletionDate.String, note.TaskCompletionTime.String, note.NoteStatus.String, note.NoteDelegation.String, note.Owner)
	a.checkInternalServerError(err, w)

	

	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

func (a *App) updateHandler(w http.ResponseWriter, r *http.Request) {
    a.isAuthenticated(w, r)

    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    var note Note
    note.ID, _ = strconv.Atoi(r.FormValue("Id"))
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
        a.checkInternalServerError(err, w)
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
	a.checkInternalServerError(err, w)

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
	fmt.Printf("%s\n", noteID) //n	NOT GETTING PASSED
	fmt.Printf("%s\n", privileges)
	fmt.Printf("%s\n", sharedUsername)

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
        a.checkInternalServerError(err, w)
        return
    }

    if !noteExists {
        http.Error(w, "Note does not exist", http.StatusBadRequest)
        return
    }

    

    // Insert a new row into user_shares table to link the shared user with the note
    _, err = a.db.Exec(`
        INSERT INTO user_shares (note_id, username , privileges)
        VALUES ($1, $2, $3)
    `,noteID, sharedUsername, privileges)
    if err != nil {
        a.checkInternalServerError(err, w)
        return
    }

    // Provide feedback to the user (e.g., "Note shared successfully")

    // Redirect to an appropriate page
    http.Redirect(w, r, "/list", http.StatusSeeOther)
}
















func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	a.isAuthenticated(w, r)
	http.Redirect(w, r, "/list", http.StatusSeeOther)
}

