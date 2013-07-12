# Mail.go

It's Programmer Law that everyone must write a mail client. This is
mine.

An imap client, for the web.

## server design

written with
[Revel](http://robfig.github.io/revel/manual/organization.html), a
Hacker News darling framework.

Uses simple, in-memory data structures for session persistence.

Layers of abstraction, so far
    
                 Server       Mailbox       Message
    models:   ----------------------------------------
                   imap.Client        tls.Config    
    
We can leverage the first-party crypto extension library `go.crypto` for
[PGP support](http://godoc.org/code.google.com/p/go.crypto/openpgp)
    
## Client design

Mobile-first design, relying on Zurb Foundation with SCSS for styling.

Coffeescript.

Front-end JS frameworks ideas:

- aura.js :: http://aurajs.github.io/aura/
- Polymer if I'm feeling especially futuristic

possible compnents/widgets:
    
    - Mailbox preview widget
    - Message widget
        - message header
        - message footer
        - message body

## Future TODOs

- use notmuch for message indexing, search, and tags: ::
  http://notmuchmail.org/

- send mail via `sendmail`
- provide a GnuPG email interface that isn't total shit
    - it really shouldn't be too hard to imporve on the current things
      here
    - maybe this would be better suited to a seperate project
