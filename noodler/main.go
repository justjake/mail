// Noodler -- test out models and stuff or something
package main

import (
    "fmt"

    // my code
    _ "github.com/justjake/mail/app/models"
    "github.com/justjake/mail/app/assets"

    "code.google.com/p/gopass"
    "code.google.com/p/go-imap/go1/imap"
    "time"
    "net/mail"
    "bytes"
    "os"
    
    // cert stuff. fuck this
    "crypto/x509"
    "crypto/tls"
)

const Hostname   = "hal.rescomp.berkeley.edu"
const Username   = "jitl@rescomp.berkeley.edu"
const SessionKey = "jake"

/*
func main() {
    print("ok")

    sess := models.NewOrCreateSession("jake")

    password, _ := gopass.GetPass("IMAP password")
    

    server := &models.Server{
        Hostname: Hostname,
        Username: "jitl@rescomp.berkeley.edu",
        Password: password,
    }

    sess[Hostname] = server

    // connect
    c, err := server.Connect()
    fmt.Printf("Connect attempt: %v, err: %v\n", c, err)

}
*/

func fatal(zone string, err error) {
    if err != nil {
        fmt.Printf("Error at '%v': %v\n", zone, err)
        os.Exit(1)
    }
}

func tls_config() *tls.Config {
    rescomp_cert_bytes := assets.LoadRescompCA()

    cert, err := x509.ParseCertificate(rescomp_cert_bytes)
    fatal("ca cert parse", err)

    pool := x509.NewCertPool()
    pool.AddCert(cert)

    return &tls.Config{ RootCAs: pool }
}

func main() {
    //
    // Note: most of error handling code is omitted for brevity
    //
    var (
        c   *imap.Client
        cmd *imap.Command
        rsp *imap.Response
    )

    Password, _ := gopass.GetPass("IMAP password> ")

    // Connect to the server
    c, err := imap.DialTLS(Hostname, tls_config())
    fatal("dial", err)


    // Remember to log out and close the connection when finished
    defer c.Logout(30 * time.Second)

    // Print server greeting (first response in the unilateral server data queue)
    fmt.Println("Server says hello:", c.Data[0].Info)
    c.Data = nil

    // Enable encryption, if supported by the server
    if c.Caps["STARTTLS"] {
        c.StartTLS(nil)
    }

    // Authenticate
    if c.State() == imap.Login {
        c.Login(Username, Password)
    }

    // List all top-level mailboxes, wait for the command to finish
    cmd, _ = imap.Wait(c.List("", "%"))

    // Print mailbox information
    fmt.Println("\nTop-level mailboxes:")
    for _, rsp = range cmd.Data {
        fmt.Println("|--", rsp.MailboxInfo())
    }

    // Check for new unilateral server data responses
    for _, rsp = range c.Data {
        fmt.Println("Server data:", rsp)
    }
    c.Data = nil

    // Open a mailbox (synchronous command - no need for imap.Wait)
    c.Select("INBOX", true)
    fmt.Print("\nMailbox status:\n", c.Mailbox)

    // Fetch the headers of the 10 most recent messages
    set, _ := imap.NewSeqSet("")
    if c.Mailbox.Messages >= 10 {
        set.AddRange(c.Mailbox.Messages-9, c.Mailbox.Messages)
    } else {
        set.Add("1:*")
    }
    cmd, _ = c.Fetch(set, "RFC822.HEADER")

    // Process responses while the command is running
    fmt.Println("\nMost recent messages:")
    for cmd.InProgress() {
        // Wait for the next response (no timeout)
        c.Recv(-1)

        // Process command data
        for _, rsp = range cmd.Data {
            header := imap.AsBytes(rsp.MessageInfo().Attrs["RFC822.HEADER"])
            if msg, _ := mail.ReadMessage(bytes.NewReader(header)); msg != nil {
                fmt.Println("|--", msg.Header.Get("Subject"))
            }
        }
        cmd.Data = nil

        // Process unilateral server data
        for _, rsp = range c.Data {
            fmt.Println("Server data:", rsp)
        }
        c.Data = nil
    }

    // Check command completion status
    if rsp, err := cmd.Result(imap.OK); err != nil {
        if err == imap.ErrAborted {
            fmt.Println("Fetch command aborted")
        } else {
            fmt.Println("Fetch error:", rsp.Info)
        }
    }
}
