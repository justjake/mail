package models

import (
    "code.google.com/p/go-imap/go1/imap"
    "net/mail"
    "container/list"
    "bytes"
    "io"
    "io/ioutil"
    "fmt"
    "mime/multipart"
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
// Mesage
// represents a single email
type Message struct {
    UID       uint32
    server    *Server
    mailbox   *Mailbox
    Header    mail.Header
    BodyData  []byte    `json:"-"` // field ignored in JSON -- use Body
                                   // instead
    Body      []*Part      // populated with MIB body sections or just
                           // body text
}




// messages are usually created with just header information
// this method downloads the actual body of the message from the server,
// optionally marking the message as '\Seen', in IMAP terms.
func (m *Message) RetrieveBody(setRead bool) (body []byte, err error) {
    // cache
    if m.BodyData != nil {
        return m.BodyData, nil
    }

    // what will our FETCH request?
    var requestType string
    if setRead {
        requestType = "BODY[TEXT]"
    } else {
        requestType = "BODY.PEEK[TEXT]"
    }


    c, err := m.server.Connect()
    if err != nil { return }

    // fetch message by UID
    set, err := imap.NewSeqSet(fmt.Sprintf("%d", m.UID))
    if err != nil { return }
    cmd, err := imap.Wait(c.UIDFetch(set, requestType))
    if err != nil { return }

    // save response data to struct
    rsp := cmd.Data[0]
    info := rsp.MessageInfo()
    m.BodyData = imap.AsBytes(info.Attrs["BODY[TEXT]"])
    return m.BodyData, nil
}

///////////////////////////////////////////////////////////////////////////////
// Part
// basically an atattchment in the email
type Part struct {
    MimeType string
    Data     []byte
}

// parse the raw RFC822.BODY bytes into seperate attatchment pieces
// if the body is not a multi-part body, this will still return a
// lenght-one slice of parts.
// this is how you get parts.
func (m *Message) GetParts() (parts []*Part, err error) {
    // can't parse the body unless we have it
    if m.BodyData == nil {
        return nil, fmt.Errorf("Cannot parse nil body")
    }

    content_type := m.Header.Get("Content-Type")
    boundry := m.Header.Get("boundry")

    // only parse multipart/alternative types
    if content_type == "multipart/alternative" {
        if boundry == "" {
            return nil, fmt.Errorf("Boundry was an empty string")
        }

        rdr := multipart.NewReader(bytes.NewReader(m.BodyData), boundry)
        parts := make([]*Part, 2)

        // loop until we reach the end of the multi-part message
        for part, err := rdr.NextPart(); err != io.EOF; part, err = rdr.NextPart() {
            if err != nil {
                return parts, fmt.Errorf("Error while parsing multipart mail: %v", err)
            }

            // read all of the part's body data into data
            data, err := ioutil.ReadAll(part)
            if err != nil { return parts, err }

            // save everything in the part
            my_part := &Part{part.Header.Get("Content-Type"), data }
            parts = append(parts, my_part)
        }
        return parts, nil
    }

    // base case: not a multipart message
    // still, read the body as a byte array and encode its content-type
    if content_type == "" { content_type = "text/plain" }
    singular_part := &Part{content_type, m.BodyData}
    return []*Part{singular_part}, nil
}
