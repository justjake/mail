package controllers

import (
    "github.com/robfig/revel"
    "github.com/justjake/mail/app/models"
)

const CurrentServerKey = "__CURRENT__"

type Mailboxes struct {
    *revel.Controller
}

// datatypes for providing JSON mailbox listings to the client
type mailboxList struct {
    Mailboxes []*models.Mailbox
    Hostname  string
}

type mailList struct {
    Messages  []*models.Message
    Hostname  string
    Mailbox   string
}



// get the current server or redirect the user to the server config page
func (c Mailboxes) getCurrentServer() (*models.Server, revel.Result) {
    session := models.GetSession(c.Session.Id())
    current_server, ok := session[CurrentServerKey]

    // no current server, which means the user has no servers
    // send them to the server list to pick and/or add one
    if !ok {
        c.Flash.Error("You must configure a server")
        return nil, c.Redirect(Servers.Index)
    }

    return current_server, nil
}

// list mailboxes on the current server
func (c Mailboxes) Index() revel.Result {

    current_server, redirect := c.getCurrentServer()
    if redirect != nil { return redirect }

    boxes, err := current_server.GetMailboxes()
    if err != nil {
        return c.RenderError(err)
    }

    res := &mailboxList{boxes, current_server.Hostname}

    // just return JSON for now
    return c.RenderJson(res)
}

func (c Mailboxes) Messages(box string) revel.Result {

    // get the current server that we will look for the mailbox in
    current_server, redirect := c.getCurrentServer()
    if redirect != nil { return redirect }

    // create mailbox here
    mbox := models.NewMailbox(box, current_server)

    // try an upper-date this hur mailbox
    messages, err := mbox.Update()
    if err != nil {
        return c.RenderError(err)
    }

    // store the mailbox, for old time's sake
    current_server.Mailboxes[box] = mbox

    // return our beautiful json results
    result := &mailList{messages, current_server.Hostname, box}
    return c.RenderJson(result)
}

func (c Mailboxes) ShowMessage(box string, uuid uint32) revel.Result {
    // TODO: finish
    return c.Render()
}

