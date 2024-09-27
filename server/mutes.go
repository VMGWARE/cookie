package server

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/discuitnet/discuit/core"
	"github.com/discuitnet/discuit/internal/httperr"
	"github.com/discuitnet/discuit/internal/uid"
)

// @Summary		Get a list of muted users and communities
// @Description	Get a list of muted users and communities
// @Router			/api/mutes [GET]
// @Success		200
// @Tags			Mutes
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) getListMutedUsersCommunities(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	writeMutes := func(w io.Writer) error {
		commMutes, err := core.GetMutedCommunities(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}
		userMutes, err := core.GetMutedUsers(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}

		if commMutes == nil {
			commMutes = []*core.Mute{}
		}
		if userMutes == nil {
			userMutes = []*core.Mute{}
		}

		response := struct {
			CommunityMutes []*core.Mute `json:"communityMutes"`
			UserMutes      []*core.Mute `json:"userMutes"`
		}{commMutes, userMutes}

		return json.NewEncoder(w).Encode(response)
	}

	switch r.req.Method {
	case "GET":
		if err := writeMutes(w); err != nil {
			return err
		}
	case "POST":
		request := struct {
			UserID      uid.ID `json:"userId"`
			CommunityID uid.ID `json:"communityId"`
		}{}
		if err := r.unmarshalJSONBody(&request); err != nil {
			return err
		}
		if !request.UserID.Zero() {
			if err := core.MuteUser(r.ctx, s.db, *r.viewer, request.UserID); err != nil {
				return err
			}
		}
		if !request.CommunityID.Zero() {
			if err := core.MuteCommunity(r.ctx, s.db, *r.viewer, request.CommunityID); err != nil {
				return err
			}
		}
		if err := writeMutes(w); err != nil {
			return err
		}
	case "DELETE":
		muteType := core.MuteType(r.urlQueryParamsValue("type"))
		if !muteType.Valid() {
			return httperr.NewBadRequest("invalid_mute_type", "Invalid mute type.")
		}
		buf := bytes.Buffer{} // Output saved here for now.
		if err := writeMutes(&buf); err != nil {
			return err
		}
		if err := core.ClearMutes(r.ctx, s.db, *r.viewer, muteType); err != nil {
			return err
		}
		if _, err := io.Copy(w, &buf); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_http_method", "Unsupported HTTP method.")
	}

	return nil
}

// @Summary		Mute a user or community
// @Description	Mute a user or community
// @Router			/api/mutes [POST]
// @Success		200
// @Tags			Mutes
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) muteUserOrCommunity(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	writeMutes := func(w io.Writer) error {
		commMutes, err := core.GetMutedCommunities(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}
		userMutes, err := core.GetMutedUsers(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}

		if commMutes == nil {
			commMutes = []*core.Mute{}
		}
		if userMutes == nil {
			userMutes = []*core.Mute{}
		}

		response := struct {
			CommunityMutes []*core.Mute `json:"communityMutes"`
			UserMutes      []*core.Mute `json:"userMutes"`
		}{commMutes, userMutes}

		return json.NewEncoder(w).Encode(response)
	}

	switch r.req.Method {
	case "GET":
		if err := writeMutes(w); err != nil {
			return err
		}
	case "POST":
		request := struct {
			UserID      uid.ID `json:"userId"`
			CommunityID uid.ID `json:"communityId"`
		}{}
		if err := r.unmarshalJSONBody(&request); err != nil {
			return err
		}
		if !request.UserID.Zero() {
			if err := core.MuteUser(r.ctx, s.db, *r.viewer, request.UserID); err != nil {
				return err
			}
		}
		if !request.CommunityID.Zero() {
			if err := core.MuteCommunity(r.ctx, s.db, *r.viewer, request.CommunityID); err != nil {
				return err
			}
		}
		if err := writeMutes(w); err != nil {
			return err
		}
	case "DELETE":
		muteType := core.MuteType(r.urlQueryParamsValue("type"))
		if !muteType.Valid() {
			return httperr.NewBadRequest("invalid_mute_type", "Invalid mute type.")
		}
		buf := bytes.Buffer{} // Output saved here for now.
		if err := writeMutes(&buf); err != nil {
			return err
		}
		if err := core.ClearMutes(r.ctx, s.db, *r.viewer, muteType); err != nil {
			return err
		}
		if _, err := io.Copy(w, &buf); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_http_method", "Unsupported HTTP method.")
	}

	return nil
}

// @Summary		Clear all mutes
// @Description	Clear all mutes
// @Router			/api/mutes [DELETE]
// @Success		200
// @Tags			Mutes
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			type			query	string	false	"The type of mute to clear"			Enums(user, community)
func (s *Server) clearAllMutes(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	writeMutes := func(w io.Writer) error {
		commMutes, err := core.GetMutedCommunities(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}
		userMutes, err := core.GetMutedUsers(r.ctx, s.db, *r.viewer, true)
		if err != nil {
			return err
		}

		if commMutes == nil {
			commMutes = []*core.Mute{}
		}
		if userMutes == nil {
			userMutes = []*core.Mute{}
		}

		response := struct {
			CommunityMutes []*core.Mute `json:"communityMutes"`
			UserMutes      []*core.Mute `json:"userMutes"`
		}{commMutes, userMutes}

		return json.NewEncoder(w).Encode(response)
	}

	switch r.req.Method {
	case "GET":
		if err := writeMutes(w); err != nil {
			return err
		}
	case "POST":
		request := struct {
			UserID      uid.ID `json:"userId"`
			CommunityID uid.ID `json:"communityId"`
		}{}
		if err := r.unmarshalJSONBody(&request); err != nil {
			return err
		}
		if !request.UserID.Zero() {
			if err := core.MuteUser(r.ctx, s.db, *r.viewer, request.UserID); err != nil {
				return err
			}
		}
		if !request.CommunityID.Zero() {
			if err := core.MuteCommunity(r.ctx, s.db, *r.viewer, request.CommunityID); err != nil {
				return err
			}
		}
		if err := writeMutes(w); err != nil {
			return err
		}
	case "DELETE":
		muteType := core.MuteType(r.urlQueryParamsValue("type"))
		if !muteType.Valid() {
			return httperr.NewBadRequest("invalid_mute_type", "Invalid mute type.")
		}
		buf := bytes.Buffer{} // Output saved here for now.
		if err := writeMutes(&buf); err != nil {
			return err
		}
		if err := core.ClearMutes(r.ctx, s.db, *r.viewer, muteType); err != nil {
			return err
		}
		if _, err := io.Copy(w, &buf); err != nil {
			return err
		}
	default:
		return httperr.NewBadRequest("invalid_http_method", "Unsupported HTTP method.")
	}

	return nil
}

// @Summary		Unmute a user or community
// @Description	Unmute a user or community
// @Router			/api/mutes/{muteID} [DELETE]
// @Success		200
// @Tags			Mutes
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			muteID			path	string	true	"The ID of the mute to delete"
func (s *Server) deleteMute(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	muteID := r.muxVar("muteID")
	if err := core.Unmute(r.ctx, s.db, *r.viewer, muteID); err != nil {
		return err
	}

	return w.writeString(`{"success":true}`)
}

// @Summary		Unmute a user
// @Description	Unmute a user
// @Router			/api/mutes/users/{mutedUserID} [DELETE]
// @Success		200
// @Tags			Mutes
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			mutedUserID		path	string	true	"The ID of the user to unmute"
func (s *Server) deleteUserMute(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	mutedUserID, err := strToID(r.muxVar("mutedUserID"))
	if err != nil {
		return err
	}

	if err := core.UnmuteUser(r.ctx, s.db, *r.viewer, mutedUserID); err != nil {
		return err
	}

	return w.writeString(`{"success":true}`)
}

// @Summary		Unmute a community
// @Description	Unmute a community
// @Router			/api/mutes/communities/{mutedCommunityID} [DELETE]
// @Success		200
// @Tags			Mutes
// @Param			Authorization		header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			mutedCommunityID	path	string	true	"The ID of the community to unmute"
func (s *Server) deleteCommunityMute(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	mutedCommunityID, err := strToID(r.muxVar("mutedCommunityID"))
	if err != nil {
		return err
	}

	if err := core.UnmuteCommunity(r.ctx, s.db, *r.viewer, mutedCommunityID); err != nil {
		return err
	}

	return w.writeString(`{"success":true}`)
}
