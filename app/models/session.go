// Per-user database
package models

// nice imap library
// http://godoc.org/code.google.com/p/go-imap/go1/imap
import "code.google.com/p/go-imap/go1/imap"
import "time"

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


///////// server ///////////
// connection to an IMAP server
// totally in-memory

const ServerLogoutTimeout = 10 * time.Second

type Server struct {
    Hostname string
    Username string
    Password string
    client *imap.Client
}

// esablish an IMAP connection
// TODO: establish TLS with sensible tls.Config (is this required? example uses nil :\ )
func (s *Server) Connect() (*imap.Client, error) {
    // re-use existing connection
    if s.client != nil {
        return s.client, nil
    }

    // establish new connection
    c, err := imap.Dial(s.Hostname)
    if err != nil { return nil, err }

    s.client = c

    // enable encryption if supported
    if c.Caps["STARTTLS"] {
        _, err := c.StartTLS(nil)
        if err != nil { 
            s.Close() 
            return nil, err
        }
    }

    // log in
    if c.State() == imap.Login {
        _, err := c.Login(s.Username, s.Password)
        if err != nil {
            s.Close()
            return nil, err
        }
    }

    return client, nil
}

func (s *Server) Close() (error) {
    if s.client != nil {
        _, err := s.client.Logout(ServerLogoutTimeout)
        return err
    }
    s.client = nil
    return nil

}
