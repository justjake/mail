// Noodler -- test out models and stuff or something
package main

import (
    "fmt"

    // my code
    "github.com/justjake/mail/app/models"
    "encoding/json"

     "code.google.com/p/gopass"
    _ "code.google.com/p/go-imap/go1/imap"
    _ "time"
    _ "net/mail"
    _ "bytes"
    "os"
    _ "io/ioutil"
    _ "strings"
    
    // cert stuff. fuck this
    "bytes"
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

func test_marshalreader() {
    // test MarshalReader
    test_str := "hello world."
    rdr := bytes.NewReader([]byte(test_str))


    mr := models.NewMarshalReader(rdr)
    data, err := mr.Data()
    fmt.Printf("MarshalReader basic test: \ndata: %d\nerror: %v\n", data, err)

    newer, backup := models.BackupReader(mr)
    fmt.Println(newer.Data())
    fmt.Println(backup.Data())
}


func main() {
    // create our models
    sess := models.GetSession("jake")
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
    _, err = server.GetMailboxes()
    fatal("get mailboxes", err)

    // messages in spam box
    spam, ok := server.Mailboxes["INBOX"]
    if !ok {
        fmt.Println("Couldn't get mailbox 'spam'")
        os.Exit(1)
    }

    msgs, err := spam.Update()
    fatal("update spam", err)

    lastMsg := msgs[ len(msgs) - 1 ]

    // download and parse body
    _, err = lastMsg.Body(false)
    fatal("download last message", err)

    // fmt.Printf("body:::::::\n%v\n", string(body))


    msg_tree, err := lastMsg.ParseBody()
    fatal("parse body", err)

    ///var recurseNode func(node *models.MessageNode)
    ///recurseNode = func (node *models.MessageNode) {
       ////fmt.Printf("node %v\n", node)
       ////if node.Children != nil {
           ////for _, child := range node.Children {
               ////recurseNode(child)
           ////}
       ////}
    ///}

    ///recurseNode(msg_tree)

    fmt.Printf("Message tree --\n%v\n", msg_tree)

    indented, err := json.MarshalIndent(msg_tree, "derp", "    ")
    fmt.Println(string(indented))



    // investigate all the different FETCH types
    // see http://tools.ietf.org/html/rfc3501#section-6.4.5
    // fetch_types := []string{
    //     "BODYSTRUCTURE",
    //     // "FLAGS", "INTERNALDATE", "RFC822.SIZE", "ENVELOPE", "BODY",
    //     // "BODYSTRUCTURE", "RFC822", "RFC822.HEADER",
    //     // "ENVELOPE", "INTERNALDATE",
    // }

    // req_type := strings.Join(fetch_types, " ")

    // cmd, err := lastMsg.RetrieveRaw(req_type)
    // fatal("request lots of things async", err)

    // cmd, err = imap.Wait(cmd, err)
    // fatal("request lots of things sync", err)

    // info := cmd.Data[0].MessageInfo()

    // for fetch_type, response := range info.Attrs {
    //     fmt.Printf("\n\n___FETCHTYPE[ %s ]___\n%s", fetch_type, imap.AsString(response))
    // }

    // retrieve and print body of final msg
    //_, err = lastMsg.RetrieveBody(false)
    //fatal("get body", err)

    //parts, err := lastMsg.GetParts()
    //fatal("get parts", err)

    //for i, p := range parts {
    //    fmt.Printf("__PART___ Number: %d\nContent-Type: %s\n\n%s", i, p.MimeType, string(p.Data))
    //}

    // close connection
    server.Close()
}
