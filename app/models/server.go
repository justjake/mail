// the business of email is this package's business
package models

import (
    // nice imap library
    // http://godoc.org/code.google.com/p/go-imap/go1/imap
    "code.google.com/p/go-imap/go1/imap"
    "time"
    "log"
    "crypto/tls"
    "container/list"
)

///////// server ///////////
// connection to an IMAP server
// totally in-memory

// no security right now
// TODO: security exception for Rescomp only; can't get certs to verify
var TLSConfig = &tls.Config{InsecureSkipVerify: true}
// brief timeout to wait for callback when closing IMAP connections
const ServerLogoutPause = 10 * time.Second
// we disconnect from the IMAP server after this much time with no request to Connect()
const NoUsageDisconnect = 20 * time.Minute

type Server struct {
    Hostname string
    Username string
    Password string
    UseTLS   bool
    client *imap.Client
    disconnectTimer *time.Timer

    Mailboxes map[string]*Mailbox
}

func NewServer(hostname, username, password string) *Server {
    server := &Server{
        Hostname:  hostname,
        Username:  username,
        Password:  password,
        UseTLS:    true,
        Mailboxes: make(map[string]*Mailbox),
    }
    return server
}

// use whatever imap connection type is specified by s.UseTLS
func (s *Server) Connect() (*imap.Client, error) {
    if s.client != nil {
        s.disconnectTimer.Reset(NoUsageDisconnect)
        return s.client, nil
    }

    var c *imap.Client
    var err error

    // actual dailing happens
    if s.UseTLS {
        c, err = s.dialTLS()
    } else {
        c, err = s.dial()
    }
    if err != nil { return nil, err }

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

// esablish an IMAP connection over TLS
func (s *Server) dialTLS() (*imap.Client, error) {
    // establish new connection
    c, err := imap.DialTLS(s.Hostname, TLSConfig)
    if err != nil { return nil, err }

    s.client = c

    return c, nil
}



// esablish an IMAP connection and upgrade to TLS if possible via STARTTLS
func (s *Server) dial() (*imap.Client, error) {
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
        log.Printf("Connection %v: TLS DISABLED\n", c)
    }

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

// geet all the top-level mailboxes in the server, and return them 
func (s *Server) GetMailboxes() (boxes []*Mailbox, err error) {
    c, err := s.Connect()
    if err != nil { return }

    // fetch data synchronously
    cmd, err := imap.Wait(c.List("", "%"))
    if err != nil { return }

    boxes = make([]*Mailbox, len(cmd.Data))

    for i, rsp := range cmd.Data {
        info := rsp.MailboxInfo()
        mbox := &Mailbox{
            Name: info.Name,
            Messages: make(map[uint32]*Message),

            server: s,
            messageUIDs: list.New(),
        }
        boxes[i] = mbox
        s.Mailboxes[mbox.Name] = mbox
    }

    return boxes, nil
}

