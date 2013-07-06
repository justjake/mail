package controllers

import (
    "github.com/robfig/revel"
)

type Mailbox struct {
    *revel.Controller
}

// list all mailboxes across open server connections
func (m Mailbox) Index() revel.Result {

}
