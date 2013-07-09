package controllers

import (
    "github.com/robfig/revel"
    "github.com/justjake/mail/app/models"
)

type Servers struct {
    *revel.Controller
}

// list the currently-added servers
//
// allow the user to activate one of the servers and store it in 
// the session under CurrentServerKey
// 
// allow the user to add a new server by posting a message to
// /servers/add with hostname, username, password, and UseTLS(bool)
func (c Servers) Index() revel.Result {

    session := models.GetSession(c.Session.Id())
    server_count := len(session)

    cur, ok := session[CurrentServerKey]
    if ok {
        // duplicate
        server_count -= 1
    }

    // enumerate servers into a nice list
    servers := make([]*models.Server, server_count)
    for hn, server := range session {
        if hn == CurrentServerKey { continue }
        servers = append(servers, server)
    }

    return c.Render(servers, cur)
}

// accepts a post to create a new server
func (c Servers) Add(hostname, username, password string, useTLS bool) revel.Result {
    // make sure we have the big-3 data we need to connect to a server
    // tls we will assume is False if it is unspecified
    c.Validation.Required(hostname).Message("You must specify a server to add.")
    c.Validation.Required(username).Message("You must supply an email address to add a server.")
    c.Validation.Required(password)

    // redirect on error
    if c.Validation.HasErrors() {
        c.Validation.Keep()
        c.FlashParams()
        return c.Redirect(Servers.Index)
    }

    // create the server
    server := models.NewServer(hostname, username, password)
    server.UseTLS = useTLS

    // test connection
    _, err := server.Connect()
    if err != nil {
        c.Flash.Error("Connection to %s failed: %v", hostname, err)
        return c.Redirect(Servers.Index)
    }

    // cool, save the server in the session
    session := models.GetSession(c.Session.Id())
    session[hostname] = server
    // also make it the new current server
    session[CurrentServerKey] = server

    c.Flash.Success("Added server %s!", hostname)
    return c.Redirect(Servers.Index)
}

func (c Servers) Select(hostname string) revel.Result {
    // needs hostname desperatley
    c.Validation.Required(hostname).Message("You must select a host.")
    if c.Validation.HasErrors() {
        c.Validation.Keep()
        c.FlashParams()
        return c.Redirect(Servers.Index)
    }

    session := models.GetSession(c.Session.Id())

    if server, ok := session[hostname]; ok {
        session[CurrentServerKey] = server
        c.Flash.Success("Selected server %s.", hostname)
        return c.Redirect(Servers.Index)
    }

    // base case -- we don't have this server
    c.Flash.Error("Server %s cannot be selected because no connection to it exists.",
        hostname)
    return c.Redirect(Servers.Index)
}

func (c Servers) Remove(hostname string) revel.Result {
    c.Validation.Required(hostname).Message("You must specify a host to remove.")
    if c.Validation.HasErrors() {
        c.Validation.Keep()
        c.FlashParams()
        return c.Redirect(Servers.Index)
    }

    session := models.GetSession(c.Session.Id())

    if server, ok := session[hostname]; ok {
        // make sure to delete second reference if this is the current
        // server
        if session[CurrentServerKey] == server {
            delete(session, CurrentServerKey)
        }
        // always close connections
        server.Close()
        delete(session, hostname)

        c.Flash.Success("Removed server %s and disconnected.", hostname)
        return c.Redirect(Servers.Index)
    }

    c.Flash.Error("Server %s not found for this session.", hostname)
    return c.Redirect(Servers.Index)
}





