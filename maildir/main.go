// Maildir parsing and handling in Go
// see http://cr.yp.to/proto/maildir.html
// see http://wiki2.dovecot.org/MailboxFormat/Maildir
package maildir


import (
    "fmt"
    "os"
    "net/mail"  // oh heavens this is wonderful
                // mail in the standard lib!
    "io"
    "io/ioutil"
    pathlib  "path" 

    "mime"
    "mime/multipart"
)


// Library configuration

// length of initial slice we allocate when finding sub-directories in
// Directory objects
const ExpectedSubdirectories uint = 5

// number of sub-messages to expect in multi-part messages
const ExpectedMessageParts uint = 5


// maildir flags
type Flag uint8

const (
    _ = iota
    Passed  Flag = 1 << iota  // P
    Replied                   // R
    Seen                      // S
    Trashed                   // T
    Draft                     // D
    Flagged                   // F
)



// hash lookups
const ContentType string = "Content-Type"
const TypeMultipart string = "multipart/alternative"
const ParamBoundry string = "boundry"

type Directory struct {
    Path string
    messageList []string // caching
    dirList     []string
}



// convert a single-character Maildir flag into data
func ReadFlag(f rune) Flag {
    switch f {
    case 'P': return Passed
    case 'R': return Replied
    case 'S': return Seen
    case 'T': return Trashed
    case 'D': return Draft
    case 'F': return Flagged
    }
    return 0
}


// checks that path is indeed a maildir, and pre-populates the sub-folder
// list
func NewDirectory(path string) (*Directory, error) {
    infos, err := ioutil.ReadDir(path)
    if err != nil { return nil, err }

    hasNew, hasCur, hasTmp := false, false, false
    dirlist := make([]string, ExpectedSubdirectories)
    for _, f := range infos {
        if f.IsDir() {

            // make sure we see the three maildir required directories
            switch f.Name() {
            case "new":
                hasNew = true
                continue
            case "cur":
                hasCur = true
                continue
            case "tmp":
                hasTmp = true
                continue
            }
            
            // add any other directories to the dirlist
            dirlist = append(dirlist, pathlib.Join(path, f.Name()))
        }

    }

    if hasNew && hasCur && hasTmp {
        return &Directory{
            Path: path,
            dirList: dirlist,
            messageList: nil,
        }, nil
    } else {
        return nil, fmt.Errorf("Folder %v does not contain 'new', 'cur', and 'tmp'.", path)
    }
}


// retrieve the name of the maildir
func (d Directory) Name() string {
    return pathlib.Base(d.Path)
}

// get the paths to every message in the maildir
func (d Directory) Messages() ([]string, error) {
    for _, subdir := range [2]string{"new", "cur"} {
        path := pathlib.Join(d.Path, subdir)
        _ = path
        // TODO: finish
    }

    return nil, fmt.Errorf("Not implemented")
}

// get the paths to every folder in the maildir
func (d Directory) Folders() ([]string, error) {
    return d.dirList, nil
}

// an Email message
type Message struct {
    Path            string
    Flags           Flag

    // everything has these right
    Header          map[string][]string
    Body            io.Reader

    attatchments    []*multipart.Part  // cached
}


// returns a mail message from a path to a maildir message file
func LoadMessage(path string) (msg *Message, err error) {
    // get a file reader for the string
    file, err := os.Open(path)
    if err != nil { return nil, err }

    // TODO: read maildir flags from the filename!

    // parse using the nice, pretty standard lib. nice and pretty.
    parsed, err := mail.ReadMessage(file)
    if err != nil { return nil, err }

    // instantiate our personal mail structure
    msg = &Message{
        Path:    path,
        Header:  parsed.Header,
        Body:    parsed.Body,
    }

    return msg, nil
}

// parse multipart messages if possible
// start by determining the content type,
// then use a mime/multipart to extract all the parts
func (m *Message) Attatchments() ([]*multipart.Part, error) {
    // reuse cache
    if m.attatchments != nil {
        return m.attatchments, nil
    }

    ct, ok := m.Header[ContentType]
    if ok {
        // parse
        media, params, err := mime.ParseMediaType(ct[0])
        if err != nil { return nil, err }

        boundry, has_boundry := params[ParamBoundry]
        if media == TypeMultipart && has_boundry {
            // awesome. time to parse us some multipart
            mp := multipart.NewReader(m.Body, boundry)
            att := make([]*multipart.Part, ExpectedMessageParts)

            // loop until we reach EOF
            var re error
            var part *multipart.Part
            for re = nil; re != io.EOF; part, re = mp.NextPart() {
                if re != nil {
                    return nil, re
                }
                att = append(att, part)
            }

            // cache and return results
            m.attatchments = att
            return m.attatchments, nil
        } else {
            return nil, fmt.Errorf("Wrong media '%v' while boundry retrieval was '%v'.", media, boundry)
        }
    } else {
        return nil, fmt.Errorf("No header named '%v'", ContentType)
    }
}

