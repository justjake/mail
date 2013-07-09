// Per-user database
package models

/////// session //////////
// a mapping of a session id -> IMAP server list
type Session   map[string]*Server

// in-memory datastore
var sessions = make(map[string]Session)

// create
func GetSession(id string) Session {
    ses, ok := sessions[id]
    if ok {
        return ses
    }

    // create new session
    ses = make(Session)
    sessions[id] = ses
    return ses
}

// remove a session from the datastore
func DestroySession(id string) {
    if ses, ok := sessions[id]; ok {
        // close all connections
        for _, server := range ses {
            server.Close()
        }
    }

    delete(sessions, id)
}
