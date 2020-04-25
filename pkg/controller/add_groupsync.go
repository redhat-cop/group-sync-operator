package controller

import (
	"github.com/redhat-cop/group-sync-operator/pkg/controller/groupsync"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, groupsync.Add)
}
