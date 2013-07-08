// the business of email is this package's business
package models

import (
    // nice imap library
    // http://godoc.org/code.google.com/p/go-imap/go1/imap
    "code.google.com/p/go-imap/go1/imap"
    "time"
    "log"
)

///////// server ///////////
// connection to an IMAP server
// totally in-memory

const ServerLogoutPause = 10 * time.Second

// we disconnect from the IMAP server after this much time with no request to Connect()
const NoUsageDisconnect = 20 * time.Minute

type Server struct {
    Hostname string
    Username string
    Password string
    client *imap.Client
    disconnectTimer *time.Timer
}

// esablish an IMAP connection
// TODO: establish TLS with sensible tls.Config (is this required? example uses nil :\ )
func (s *Server) Connect() (*imap.Client, error) {
    // re-use existing connection
    if s.client != nil {
        s.disconnectTimer.Reset(NoUsageDisconnect)
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
    } else {
        log.Printf("Connection %v: TLS DISABLED", c)
    }

    // log in
    if c.State() == imap.Login {
        _, err := c.Login(s.Username, s.Password)
        if err != nil {
            s.Close()
            return nil, err
        }
    }

    // auto-disconnect after a certain timeout
    s.disconnectTimer = time.AfterFunc(NoUsageDisconnect, func () {
        s.Close()
    })

    return c, nil
}

func (s *Server) Close() (error) {
    // stop timer and nil it
    if s.disconnectTimer != nil {
        s.disconnectTimer.Stop()
        s.disconnectTimer = nil
    }

    // close server connection
    if s.client != nil {
        _, err := s.client.Logout(ServerLogoutPause)
        s.client = nil
        return err
    }

    return nil
}
