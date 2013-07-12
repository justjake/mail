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
    "strings"
)

const debug_mode = true
func debug(msgs... interface{}) {
    if debug_mode {
        fmt.Println(msgs...)
    }
}

// MIME header name to detect MIME Content-Type
const ContentType    = "Content-Type"
// MIME header name to retrieve mime/multipart boundry strings
const Boundary        = "boundary"
// MIME type of multipart messages
const TypeMultipart  = "multipart"
// Regexp that matches MIME multipart MIME types
var   MultipartRegex = regexp.MustCompile("^"+TypeMultipart)

// is a raw content-type string a multipart message?
func MultipartType(content_type string) (boundry string, ok bool) {
    mt, params, err := mime.ParseMediaType(content_type)
    if err != nil { return "", false }
    if MultipartRegex.MatchString(mt) {
        boundry, ok = params[Boundary]
        return
    }
    return "", false
}

// Prints a pretty little table of the errors, like so:
// 
//  Errors [2] encountered while converting children:
//    |  No boundry given for TypeMultipart <ponter goes here>
//    |  Errors [1] encountered while converting children:
//    |    |  Aborted Multipart#NextPart at error: <pointer goes here>
//
type ChildError map[*MessageNode]error
const childIndent= "  |  "
func (oops ChildError) Error() string {
    ret := fmt.Sprintf("Errors [%d] encountered while converting children:\n", len(oops))
    sub_errors := make([]string, len(oops))
    j := 0
    for _, err := range oops {
        // indent child errors
        lines := strings.Split(err.Error(), "\n")
        for i, line := range lines {
            lines[i] = childIndent + line
        }
        sub_errors[j] = strings.Join(lines, "\n")
        j++
    }
    return ret + strings.Join(sub_errors, "\n")
}


// a tree structure of MIME multipart messages
type MessageNode struct {
    // the MIME content-type of this part.
    // When content-type is a mutlipart type, the message node's
    // Children field will be populated
    ContentType string
    ContentTypeParams map[string]string
    Header      mail.Header

    // child message nodes, if this message node was a 'multipart' message
    Children    []*MessageNode
    // contains data only if we could not derive Children
    // its nil in the best cases <3
    Body        *MarshalReader
}

// nice to-string for debugging
const sectionSep = "----"
func (node *MessageNode) StringIndent(indent string) string {

    // new-line seperated k: v header list
    header := make([]string, len(node.Header)+2)
    header[0] = indent + sectionSep
    header[len(header) - 1] = indent + sectionSep
    i := 1
    for k, v := range node.Header {
        header[i] = fmt.Sprintf("%s: %v", indent + k, v)
        i++
    }

    if node.Children != nil {
        // recurse simialar for child nodes
        bodies := make([]string, len(node.Children) + 1)
        for i, child := range node.Children {
            bodies[i] = child.StringIndent(indent + childIndent)
        }
        return strings.Join(append(header, bodies...), "\n")
    } else {
        // 
        bodyData, err := node.Body.Data()
        if err != nil {
            debug("issue in stringIndent: error occured when doing bodyData: ", err)
        }
        bodyString := indent + strings.Replace(string(bodyData), "\n", "\n" + indent, -1)
        return strings.Join(header, "\n") + "\n" + bodyString
    }
    return strings.Join(header, "\n")
}


// convert what we expect to be the RFC 5322 data
func DataToNode(data io.Reader) (*MessageNode, error) {
    msg, err := mail.ReadMessage(data)
    if err != nil { return nil, err }

    return MessageToNode(msg)
}

// run a function on each element in this tree
func (node *MessageNode) ForEach(f func(*MessageNode)) {
    f(node)
    if node.Children != nil {
        for _, child := range node.Children {
            child.ForEach(f)
        }
    }
}

// determine via MIME type if a MessageNode is binary or not
var TextMimeTypes = map[string]bool{
    "application/json":         true,
    "application/javascript":   true,
    "application/pgp-signature": true, // normal text around Base64 signature block
}
var StartsWithText = regexp.MustCompilePOSIX("^text/")
func IsText(node *MessageNode) bool {
    res := false

    // ContentType based detection
    mt, params := node.ContentType, node.ContentTypeParams
    // check for charset param
    // it there's a charset, it must be text
    if _, ok := params["charset"]; ok { res = true }

    // if it starts with text...
    if StartsWithText.MatchString(mt) { res = true }

    // know text types
    if _, ok := TextMimeTypes[mt]; ok { res = true }

    // ContentEncoding based detection
    // TODO

    debug("is IsText text? ", mt, ": ", res)
    return res
}

// recursivley parse a multipart.Part 
// will always return a *MessageNode, even on encountering errors. You will need
// good switching code to work with message nodes
// If errors occur whilst creating the sub-tree, such errors will be 
// returned in a ChildError mapping. You may ignore such errors for the most
func MessageToNode(msg *mail.Message) (*MessageNode, error) {

    debug("starting message to node for message: ", msg)
    body, backup := backupReader(msg.Body)

    node := &MessageNode{
        Header: msg.Header,
        Body: backup,
    }

    ct, params, err := mime.ParseMediaType(msg.Header.Get(ContentType))

    if err == nil {
        node.ContentType = ct
        node.ContentTypeParams = params
        if boundry, ok := params[Boundary]; ok {
            // parse the data as a multipart!

            multi := multipart.NewReader(body, boundry)
            child_err_occured := false
            child_errs := make(ChildError)
            node.Children = make([]*MessageNode, 0, 5)

            // read new Parts off the multipart parser
            for part, err := multi.NextPart(); err != io.EOF; part, err = multi.NextPart() {
                // check errors
                if err != nil {
                    debug("error occured while making part: ", part, " error: ", err)
                    // restore backup and abort parsing
                    return node, fmt.Errorf("Aborted Multipart#NextPart at error: %v", err)
                }

                // create message so we can recurse
                sub_msg := &mail.Message {
                    Header: mail.Header(part.Header),
                    Body:   NewMarshalReader(part),
                }

                // create child node
                child, err := MessageToNode(sub_msg)

                // store any errors
                if err != nil {
                    child_err_occured = true
                    child_errs[child] = err
                }

                // store child
                node.Children = append(node.Children, child)
            } // end enumerate parts

            // don't need this data anymore, since we have the children
            node.Body = nil

            if child_err_occured {
                return node, child_errs
            }
        } // end has-boundry-parse-sections
    } // end decoded-mime-type

    if node.Body != nil {
        // make sure body has text export type
        // so the browser doesn't have to do MIME detection
        node.Body.MarshalAsString = IsText(node)
    }
    return node, nil
}


