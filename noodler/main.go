// Noodler -- test out models and stuff or something
package main

import (
    "fmt"

    // my code
    "github.com/justjake/mail/app/models"

    "code.google.com/p/gopass"
    _ "code.google.com/p/go-imap/go1/imap"
    _ "time"
    _ "net/mail"
    _ "bytes"
    "os"
    
    // cert stuff. fuck this
    "crypto/tls"
)

const Hostname   = "hal.rescomp.berkeley.edu"
const Username   = "jitl@rescomp.berkeley.edu"
const SessionKey = "jake"

/*
func main() {

}
*/

func fatal(zone string, err error) {
    if err != nil {
        fmt.Printf("Error at '%v': %v\n", zone, err)
        os.Exit(1)
    }
}

func tls_config() *tls.Config {
    return &tls.Config{ 
        InsecureSkipVerify: true,
    }
}

func main() {
    // create our models
    sess := models.NewOrCreateSession("jake")
    password, err := gopass.GetPass("IMAP password> ")
    fatal("get password", err)

    server := models.NewServer(
        Hostname,
        "jitl@rescomp.berkeley.edu",
        password)

    sess[Hostname] = server

    // connect - to test
    _, err = server.Connect()
    fatal("server.Connect", err)

    // get mailboxes
    boxes, err := server.GetMailboxes()
    fatal("get mailboxes", err)

    // list boxes
    for i, box := range boxes {
        fmt.Printf("Mailbox %d: %s\n", i, box.Name)
    }

    // messages in spam box
    spam, ok := server.Mailboxes["spam"]
    if !ok {
        fmt.Println("Couldn't get mailbox 'spam'")
        os.Exit(1)
    }

    msgs, err := spam.Update()
    fatal("update spam", err)

    for e := msgs.Front(); e != nil; e = e.Next() {
        // operate on e.Value
        message := e.Value.(*models.Message)
        fmt.Printf("Subject: %s\n", message.Header.Get("Subject"))
    }

    // close connection
    server.Close()
}
