package models

import (
    "code.google.com/p/go-imap/go1/imap"
    "net/mail"
    "bytes"
    "fmt"
)

// mailbox + message, ties back to server\
// vinod has suggested always using UIDs.
// so we'll do that
type Mailbox struct {
    server        *Server
    latestMessage uint32

    Name string
    Mail map[uint32]*Email
}

// create a new Mailbox model
func NewMailbox(name string, server *Server) *Mailbox {
    return &Mailbox{
        server: server,
        latestMessage: 1,
        Name: name,
        Mail: make(map[uint32]*Email),
    }
}

// gets all the messages on the server since the last message in the list
func (m *Mailbox) Update() (newMail []*Email, err error) {
    c, err := m.server.Connect()
    if err != nil { return nil, err }

    // sync imap command to select the mailbox for actions
    c.Select(m.Name, true)

    lastHad := m.latestMessage

    // retrieve items
    wanted := fmt.Sprintf("%d:*", lastHad)
    set, err := imap.NewSeqSet(wanted)
    if err != nil { return nil, err }
    cmd, err := c.UIDFetch(set, "RFC822.HEADER UID")
    if err != nil { return nil, err }

    // result
    newMail = make([]*Email, 0, 5)

    for cmd.InProgress() {
        // Wait for the next response (no timeout)
        c.Recv(-1)

        // Process command data

        // retrieve message UID
        // construct local Message structure from given header
        // store message in map
        // append UID to newMessages list
        for _, rsp := range cmd.Data {
            info := rsp.MessageInfo()
            if info.Attrs["UID"] != nil {
                // construct message
                header := imap.AsBytes(info.Attrs["RFC822.HEADER"])
                // TODO: catch this error
                if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
                    // we could read the message and retrieve the UID
                    // so this is valid to push into our storage system
                    m.latestMessage = info.UID

                    my_msg := &MessageNode{
                        Header: msg.Header,
                        ContentType: msg.Header.Get(ContentType),
                    }

                    email := &Email {
                        server: m.server,
                        mailbox: m,
                        UID: info.UID,
                        Message: my_msg,
                    }

                    // store
                    newMail = append(newMail, email)
                    m.Mail[info.UID] = email
                } else {
                    fmt.Printf("mail.ReadMessage failed on UID %d\n", info.UID)
                }
            } else {
                fmt.Printf("Message %v had no UID. Skipped.\n", info)
            }
        }

        // clear data
        cmd.Data = nil
    }

    return newMail, nil
}


///////////////////////////////////////////////////////////////////////////////
// represents a single email
type Email struct {
    UID       uint32
    server    *Server
    mailbox   *Mailbox
    Message   *MessageNode
    bodyData  []byte
}

// Issue a FETCH request for this message
// TODO make private, this is an abstraction-breaker
func (m *Email) RetrieveRaw(requestType string) (cmd *imap.Command, err error) {
    c, err := m.server.Connect()
    if err != nil { return }

    // fetch message by UID
    set, err := imap.NewSeqSet(fmt.Sprintf("%d", m.UID))
    if err != nil { return }
    cmd, err = c.UIDFetch(set, requestType)
    return cmd, err
}


// messages are usually created with just header information
// this method downloads the actual body of the message from the server,
// optionally marking the message as '\Seen', in IMAP terms.
func (m *Email) Body(setRead bool) (body []byte, err error) {
    // cache
    if m.bodyData != nil {
        return m.bodyData, nil
    }

    // what will our FETCH request?
    var requestType string
    if setRead {
        requestType = "BODY[TEXT]"
    } else {
        requestType = "BODY.PEEK[TEXT]"
    }

    cmd, err := m.RetrieveRaw(requestType)
    cmd, err = imap.Wait(cmd, err)
    if err != nil { return }

    info := cmd.Data[0].MessageInfo()
    m.bodyData = imap.AsBytes(info.Attrs["BODY[TEXT]"])
    return m.bodyData, nil
}

// parse the raw RFC822.BODY bytes into seperate attatchment pieces
// if the body is not a multi-part body, this will still return a
// lenght-one slice of parts.
// this is how you get parts.
func (m *Email) ParseBody() (*MessageNode, error) {
    // can't parse the body unless we have it
    if m.bodyData == nil {
        return nil, fmt.Errorf("Cannot parse nil body")
    }

    msg := &mail.Message {
        Header: m.Message.Header,
        Body: bytes.NewReader(m.bodyData),
    }

    node, err := MessageToNode(msg)
    if err != nil {
        if _, ok := err.(ChildError); ok {
            // mostly-good node, keep it
            m.Message = node
            return node, err
        }
        return nil, err
    }

    return node, nil
}
