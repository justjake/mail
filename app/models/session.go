// Per-user database
package models

/////// session //////////
// a mapping of a session id -> IMAP server list
type SessionID string
type Session   map[string]*Server

// in-memory datastore
var sessions = make(map[SessionID]Session)

// create
func NewOrCreateSession(id SessionID) Session {
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
func DestroySession(id SessionID) {
    if ses, ok := sessions[id]; ok {
        // close all connections
        for _, server := range ses {
            server.Close()
        }
    }

    delete(sessions, id)
}
