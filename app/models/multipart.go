// mail utilities
package models

// this file wants to recursivley parse RFC 2046 messages

import (
    "fmt"
    "regexp"
    "mime"
    "mime/multipart"
    "net/mail"
    "io"
    "bytes"
    "strings"
)

const ContentType    = "Content-Type"
const Boundry        = "Boundry"
const TypeMultipart  = "multipart"
var   MultipartRegex = regexp.MustCompile("^"+TypeMultipart)

// is a raw content-type string a multipart message?
func IsMultipartType(content_type string) bool {
    mt, _, err := mime.ParseMediaType(content_type)
    if err != nil { return false }
    return MultipartRegex.MatchString(mt)
}

// Prints a pretty little table of the errors, like so:
// 
//  Errors [2] encountered while converting children:
//    |  No boundry given for TypeMultipart <ponter goes here>
//    |  Errors [1] encountered while converting children:
//    |    |  Aborted Multipart#NextPart at error: <pointer goes here>
//
type ChildError map[*MessageNode]error
const childErrorIndent = "  |  "
func (oops ChildError) Error() string {
    ret := fmt.Sprintf("Errors [%d] encountered while converting children:\n", len(oops))
    sub_errors := make([]string, len(oops))
    j := 0
    for _, err := range oops {
        // indent child errors
        lines := strings.Split(err.Error(), "\n")
        for i, line := range lines {
            lines[i] = childErrorIndent + line
        }
        sub_errors[j] = strings.Join(lines, "\n")
        j++
    }
    return ret + strings.Join(sub_errors, "\n")
}

// subtly duplicate a reader if you're worried
// use the `use_instead` reader for your dangerous operation
// if things go wrong, you have the `backup` to return to.
func backupReader(in_danger io.Reader) (use_instead io.Reader, backup io.Reader) {
    var already_read *bytes.Buffer

    // already_read = f - unread
    // backup = already_read + unread
    use_instead = io.TeeReader(in_danger, already_read)
    backup = io.MultiReader(already_read, in_danger)
    return use_instead, backup
}


// a tree structure of MIME multipart messages
type MessageNode struct {
    // the MIME content-type of this part.
    // When content-type is a mutlipart type, the message node's
    // Children field will be populated
    ContentType string
    Header      mail.Header

    // child message nodes, if this message node was a 'multipart' message
    Children    []*MessageNode
    // contains data only if we could not derive Children
    // its nil in the best cases <3
    Body        io.Reader
}

// convert what we expect to be the RFC 5322 data
func DataToNode(data io.Reader) (*MessageNode, error) {
    msg, err := mail.ReadMessage(data)
    if err != nil { return nil, err }

    return MessageToNode(msg)
}

// recursivley parse a multipart.Part 
func MessageToNode(msg *mail.Message) (*MessageNode, error) {
    node := &MessageNode{
        ContentType: msg.Header.Get(ContentType),
        Header: msg.Header,
        Body: msg.Body,
    }

    if IsMultipartType(node.ContentType) {
        boundry := node.Header.Get(Boundry)
        if boundry == "" {
            // nothing more to do, but invalid ContentType
            return node, fmt.Errorf("No boundry given for TypeMultipart %s", node.ContentType)
        }

        // checkpoint our reader
        body, backup := backupReader(msg.Body)
        node.Body = backup

        // parse the data as a multipart!
        multi := multipart.NewReader(body, boundry)
        parts := make([]*multipart.Part, 0, 5)
        var err error
        var sub_part  *multipart.Part
        // read new Parts off the multipart parser
        for err = nil; err != io.EOF; sub_part, err = multi.NextPart() {
            if err != nil {
                // restore backup and abort parsing
                node.Body = backup
                return node, fmt.Errorf("Aborted Multipart#NextPart at error: %v", err)
            }
            parts = append(parts, sub_part)
        }

        // recurse nodes
        child_err_occured := false
        child_errs := make(ChildError)
        node.Children  = make([]*MessageNode, len(parts))

        for i, part := range parts {
            sub_msg := &mail.Message {
                Header: mail.Header(part.Header),
                Body:   part,
            }
            node.Children[i], err = MessageToNode(sub_msg)
            // store any errors
            if err != nil {
                child_err_occured = true
                child_errs[node.Children[i]] = err
            }
        }

        // don't need this data anymore, since we have the children
        node.Body = nil

        if child_err_occured {
            return node, child_errs
        }
    }
    return node, nil
}
