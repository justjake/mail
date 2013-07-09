package models

import (
    "code.google.com/p/go-imap/go1/imap"
    "net/mail"
    "container/list"
    "bytes"
    "io"
    "fmt"
)

// mailbox + message, ties back to server\
// vinod has suggested always using UIDs.
// so we'll do that
type Mailbox struct {
    server      *Server
    messageUIDs *list.List // uint32

    Name string
    Messages map[uint32]*Message
}

// gets all the messages on the server since the last message in the list
func (m *Mailbox) Update() (newMessages *list.List /* *Message */, err error) {
    c, err := m.server.Connect()
    if err != nil { return nil, err }

    // sync imap command to select the mailbox for actions
    c.Select(m.Name, true)

    var lastHad uint32
    last := m.messageUIDs.Back()
    if last == nil {
        lastHad = 1
    } else {
        lastHad = last.Value.(uint32)
    }

    // retrieve items
    wanted := fmt.Sprintf("%d:*", lastHad)
    set, err := imap.NewSeqSet(wanted)
    if err != nil { return nil, err }
    cmd, err := c.UIDFetch(set, "RFC822.HEADER UID")
    if err != nil { return nil, err }

    // result
    newMessages = list.New()

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
                    m.messageUIDs.PushBack(info.UID)
                    my_msg := &Message{
                        server: m.server,
                        mailbox: m,
                        UID: info.UID,
                        Header: msg.Header,
                    }

                    // store
                    newMessages.PushBack(my_msg)
                    m.Messages[info.UID] = my_msg
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

    return newMessages, nil
}


///////////////////////////////////////////////////////////////////////////////
// Mesages
// represents a single email
// TODO: provide a way to easily parse and store multi-part bodies
type Message struct {
    UID uint32
    server *Server
    mailbox *Mailbox
    Header mail.Header
    Body   io.Reader
}

// messages are usually created with just header information
// this method downloads the actual body of the message from the server,
// optionally marking the message as '\Seen', in IMAP terms.
func (m *Message) RetrieveBody(setRead bool) (body io.Reader, err error) {
    // cache
    if m.Body != nil {
        return m.Body, nil
    }

    c, err := m.server.Connect()
    if err != nil { return }

    // fetch message by UID
    set, err := imap.NewSeqSet(fmt.Sprintf("%d", m.UID))
    if err != nil { return }
    cmd, err := imap.Wait(c.UIDFetch(set, "RFC822.BODY"))
    if err != nil { return }

    // turn body into an io.Reader then return
    rsp := cmd.Data[0]
    bodyBytes := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.BODY"])
    body = bytes.NewReader(bodyBytes)
    m.Body = body

    return body, nil
}
