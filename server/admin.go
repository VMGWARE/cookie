package server

import (
	"net/http"

	"github.com/discuitnet/discuit/core"
	"github.com/discuitnet/discuit/internal/httperr"
)

//	@Summary		Admin actions
//	@Description	Perform admin actions like banning users, setting default forums, etc.
//	@Router			/api/_admin [POST]
//	@Success		200
//	@Tags			Admin
//	@Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) adminActions(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	admin, err := core.GetUser(r.ctx, s.db, *r.viewer, r.viewer)
	if err != nil {
		return err
	}
	if !admin.Admin {
		return httperr.NewForbidden("not_admin", "You are not an admin.")
	}

	// User is an admin, proceed.

	reqBody, err := r.unmarshalJSONBodyToMap()
	if err != nil {
		return err
	}

	invalidJSONErr := httperr.NewBadRequest("invalid_json", "Invalid JSON body.")

	action, ok := reqBody["action"].(string)
	if !ok {
		return invalidJSONErr
	}

	switch action {
	case "ban_user":
		username, ok := reqBody["username"].(string)
		if !ok {
			return invalidJSONErr
		}
		user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
		if err != nil {
			return err
		}
		if err := s.LogoutAllSessionsOfUser(user); err != nil {
			// s.writeErrorCustom(w, r, http.StatusInternalServerError, "Error logging out user: "+err.Error(), "error_loggin_out_user")
			// return
			return &httperr.Error{
				HTTPStatus: http.StatusInternalServerError,
				Code:       "error_loggin_out_user",
				Message:    "Error logging out user: " + err.Error(),
			}
		}
		if user.Admin {
			return httperr.NewForbidden("no_ban_admin", "Admin can't ban another admin, yo!")
		}
		if _, ok := reqBody["deleteContentDays"]; ok {
			n, ok := reqBody["deleteContentDays"].(float64)
			if !ok {
				return invalidJSONErr
			}
			if err := user.DeleteContent(r.ctx, int(n), *r.viewer); err != nil {
				return err
			}
		}
		if err := user.Ban(r.ctx); err != nil {
			return err
		}
	case "unban_user":
		username, ok := reqBody["username"].(string)
		if !ok {
			return invalidJSONErr
		}
		user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
		if err != nil {
			return err
		}
		if err := user.Unban(r.ctx); err != nil {
			return err
		}
	case "add_default_forum", "remove_default_forum":
		name, ok := reqBody["name"].(string)
		if !ok {
			return invalidJSONErr
		}
		comm, err := core.GetCommunityByName(r.ctx, s.db, name, r.viewer)
		if err != nil {
			return err
		}
		if err = comm.SetDefault(r.ctx, action == "add_default_forum"); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_action", "Unsupported admin action.")
	}

	return w.writeString(`{"success:":true}`)
}
