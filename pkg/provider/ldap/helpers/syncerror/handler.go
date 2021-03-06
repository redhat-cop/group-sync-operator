package syncerror

import (
	"github.com/go-logr/logr"
	ldapquery "github.com/openshift/library-go/pkg/security/ldapquery"
)

// Handler knows how to handle errors
type Handler interface {
	// HandleError processess an error without mutating it. If the error is determined to be fatal,
	// a non-nil error should be returned.
	HandleError(err error) (handled bool, fatalError error)
}

func NewCompoundHandler(handlers ...Handler) Handler {
	return &compoundHandler{handlers: handlers}
}

// compoundHandler chains other error handlers
type compoundHandler struct {
	handlers []Handler
}

// HandleError asks each of the handlers to handle the error. If any handler decides the error is fatal or handles it,
// the chain is broken.
func (h *compoundHandler) HandleError(err error) (bool, error) {
	for _, handler := range h.handlers {
		handled, handleErr := handler.HandleError(err)
		if handled || handleErr != nil {
			return handled, handleErr
		}
	}

	return false, nil
}

func NewMemberLookupOutOfBoundsSuppressor(logger logr.Logger) Handler {
	return &memberLookupOutOfBoundsSuppressor{logger: logger}
}

// memberLookupOutOfBoundsSuppressor suppresses member lookup errors caused by a search trying to go out of the base DN bounds
type memberLookupOutOfBoundsSuppressor struct {
	logger logr.Logger
}

// HandleError suppresses member lookup errors caused by out-of-bounds queries,
func (h *memberLookupOutOfBoundsSuppressor) HandleError(err error) (bool, error) {
	memberLookupError, isMemberLookupError := err.(*memberLookupError)
	if !isMemberLookupError {
		return false, nil
	}

	if ldapquery.IsQueryOutOfBoundsError(memberLookupError.causedBy) {
		h.logger.Error(err, "Ignoring member in group", "Group", memberLookupError.ldapGroupUID, "Member", memberLookupError.ldapUserUID, "Error", memberLookupError.causedBy)
		return true, nil
	}

	return false, nil
}

func NewMemberLookupMemberNotFoundSuppressor(logger logr.Logger) Handler {
	return &memberLookupMemberNotFoundSuppressor{logger: logger}
}

// memberLookupMemberNotFoundSuppressor suppresses member lookup errors caused by a search returning no valid entries,
// which can happen in two ways:
//   - if the search is not by DN, an empty result list is returned
//   - if the search is by DN, an error is returned from the LDAP server: no such object
type memberLookupMemberNotFoundSuppressor struct {
	// err determines where a log message will be printed
	logger logr.Logger
}

// HandleError suppresses member lookup errors caused by no such object or entry not found errors,
func (h *memberLookupMemberNotFoundSuppressor) HandleError(err error) (bool, error) {
	memberLookupError, isMemberLookupError := err.(*memberLookupError)
	if !isMemberLookupError {
		return false, nil
	}

	if ldapquery.IsEntryNotFoundError(memberLookupError.causedBy) || ldapquery.IsNoSuchObjectError(memberLookupError.causedBy) {
		h.logger.Error(err, "Ignoring member in group", "Group", memberLookupError.ldapGroupUID, "Member", memberLookupError.ldapUserUID, "Error", memberLookupError.causedBy)
		return true, nil
	}

	return false, nil
}
