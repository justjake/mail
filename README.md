# Mail.go

It's Programmer Law that everyone must write a mail client. This is
mine.

A maildir reader, for the web, in Go & Coffeescript.

## server design

written with
[Revel](http://robfig.github.io/revel/manual/organization.html), a
Hacker News darling framework.

Single-user, hardcoded username/password hash authentication.

Should be comfortable running on an NFS-mounted homedir with longish
lookup times -- we musn't be writing any sort of files!

leave room for expanded functionality in the future. 

## Client design

Mobile-first design.

Coffeescript.

Front-end JS frameworks ideas:

- aura.js :: http://aurajs.github.io/aura/
- Polymer if I'm feeling especially futuristic

Widgets:
    
    - Mailbox preview widget
    - Message widget
        - message header
        - message footer
        - message body

Zurb Foundation.

## Design

URLS:

    /mailboxes/:mbox_name/
    /mailboxes/:mbox_name/:message_identifier

Configuration:

    MaildirRoot = ~/mail
    HashedPassword   = asdkfjasdfasdfasdf
    MessagesPerView  = 40
    MarkRead         = True
    
Future TODOs:

- use notmuch for message indexing, search, and tags: ::
  http://notmuchmail.org/

- send mail via `sendmail`
- provide a GnuPG email interface that isn't total shit
    - it really shouldn't be too hard to imporve on the current things
      here
    - maybe this would be better suited to a seperate project
