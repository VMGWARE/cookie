package server

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/discuitnet/discuit/core"
	"github.com/discuitnet/discuit/internal/httperr"
	"github.com/discuitnet/discuit/internal/meilisearch"
	msql "github.com/discuitnet/discuit/internal/sql"
	"github.com/discuitnet/discuit/internal/uid"
	"github.com/gorilla/mux"
)

// userModOrAdmin returns true is user is either a mod of c or an admin or both.
func userModOrAdmin(ctx context.Context, db *sql.DB, user uid.ID, c *core.Community) (bool, error) {
	if c.ViewerMod.Bool {
		return true, nil
	} else {
		user, err := core.GetUser(ctx, db, user, nil)
		if err != nil {
			return false, err
		}
		if user.Admin {
			return true, nil
		}
	}
	return false, nil
}

// @Summary		Create a community.
// @Description	Create a community.
// @Router			/api/community [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) createCommunity(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	// TODO: Limits. Fine for now, as long as no admin account is compromised.

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	name := values["name"]
	about := values["about"]
	comm, err := core.CreateCommunity(r.ctx, s.db, *r.viewer, s.config.ForumCreationReqPoints, s.config.MaxForumsPerUser, name, about)
	if err != nil {
		return err
	}

	meilisearch.CommunityUpdateOrCreateDocumentIfEnabled(r.ctx, s.config, comm)

	return w.writeJSON(comm)
}

// @Summary		Get communities.
// @Description	Get communities.
// @Router			/api/communities [GET]
// @Success		200
// @Tags			Community
// @Param			set		query	string	false	"Either 'all' or 'default' or 'subscribed'."	Enums(all,default,subscribed)
// @Param			sort	query	string	false	"Sort by 'size' or 'name'."						Enums(size,name)
func (s *Server) getCommunities(w *responseWriter, r *request) error {
	query := r.urlQueryParams()
	search := query.Get("q")

	set := query.Get("set") // Either "all" or "default" or "subscribed".
	if set == "" {
		set = core.CommunitiesSetAll
	}

	sort := core.CommunitiesSortDefault
	__sort := query.Get("sort")
	if __sort != "" {
		sort = core.CommunitiesSort(__sort)
	}

	limit := -1
	limit_str := query.Get("limit")
	if limit_str != "" {
		var err error
		limit, err = strconv.Atoi(limit_str)
		if err != nil {
			return httperr.NewBadRequest("invalid_limit", "Invalid limit.")
		}
	}

	var comms []*core.Community
	var err error

	if search != "" { // Search communities.
		comms, err = core.GetCommunitiesPrefix(r.ctx, s.db, search)
	} else {
		switch set {
		case core.CommunitiesSetAll, core.CommunitiesSetDefault:
			comms, err = core.GetCommunities(r.ctx, s.db, sort, set, limit, nil)
		case core.CommunitiesSetSubscribed:
			if !r.loggedIn {
				return errNotLoggedIn
			}
			comms, err = core.GetCommunities(r.ctx, s.db, sort, set, limit, r.viewer)
		}
	}
	if err != nil {
		if httperr.IsNotFound(err) {
			return w.writeString("[]")
		}
		return err
	}

	if r.loggedIn {
		for _, comm := range comms {
			if err = comm.PopulateViewerFields(r.ctx, *r.viewer); err != nil {
				return err
			}
		}
	}

	if len(comms) == 0 {
		return w.writeString("[]")
	}

	return w.writeJSON(comms)
}

// @Summary		Get a community.
// @Description	Get a community.
// @Router			/api/communities/{communityID} [GET]
// @Success		200
// @Tags			Community
// @Param			communityID	path	string	true	"Community ID or name."
// @Param			byName		query	string	false	"If true, communityID is treated as name."
func (s *Server) getCommunity(w *responseWriter, r *request) error {
	var (
		communityID = r.muxVar("communityID") // Community ID or name.
		query       = r.urlQueryParams()
		comm        *core.Community
		byName      = strings.ToLower(query.Get("byName")) == "true"
		err         error
	)

	if byName {
		comm, err = core.GetCommunityByName(r.ctx, s.db, communityID, r.viewer)
	} else {
		var cid uid.ID
		cid, err = uid.FromString(communityID)
		if err != nil {
			return httperr.NewBadRequest("invalid_id", "Invalid ID.")
		}
		comm, err = core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	}
	if err != nil {
		return err
	}

	if err = comm.PopulateMods(r.ctx); err != nil {
		return err
	}
	if err = comm.FetchRules(r.ctx); err != nil {
		return err
	}
	if _, err = comm.Default(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(comm)
}

// @Summary		Update a community.
// @Description	Update a community.
// @Router			/api/communities/{communityID} [PUT]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID or name."
func (s *Server) updateCommunity(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	var (
		communityID = r.muxVar("communityID") // Community ID or name.
		query       = r.urlQueryParams()
		byName      = strings.ToLower(query.Get("byName")) == "true"
		comm        *core.Community
		err         error
	)

	if byName {
		comm, err = core.GetCommunityByName(r.ctx, s.db, communityID, r.viewer)
	} else {
		var cid uid.ID
		cid, err = uid.FromString(communityID)
		if err != nil {
			return httperr.NewBadRequest("invalid_id", "Invalid ID.")
		}
		comm, err = core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	}
	if err != nil {
		return err
	}

	if err = comm.PopulateMods(r.ctx); err != nil {
		return err
	}
	if err = comm.FetchRules(r.ctx); err != nil {
		return err
	}

	rcomm := core.Community{}
	if err = r.unmarshalJSONBody(&rcomm); err != nil {
		return err
	}
	comm.NSFW = rcomm.NSFW
	comm.About = rcomm.About

	if err = comm.Update(r.ctx, *r.viewer); err != nil {
		return err
	}

	meilisearch.CommunityUpdateOrCreateDocumentIfEnabled(r.ctx, s.config, comm)

	return w.writeJSON(comm)
}

// @Summary		Join a community.
// @Description	Join a community.
// @Router			/api/_joinCommunity [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) joinCommunity(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	if err := s.rateLimit(r, "join_community_1_"+r.viewer.String(), time.Second*1, 1); err != nil {
		return err
	}
	if err := s.rateLimit(r, "join_community_2_"+r.viewer.String(), time.Hour, 500); err != nil {
		return err
	}

	req := struct {
		CommunityID uid.ID `json:"communityId"`
		Leave       bool   `json:"leave"`
	}{}
	if err := r.unmarshalJSONBody(&req); err != nil {
		return err
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	community, err := core.GetCommunityByID(r.ctx, s.db, req.CommunityID, r.viewer)
	if err != nil {
		return err
	}

	if req.Leave {
		err = community.Leave(r.ctx, user.ID)
	} else {
		err = community.Join(r.ctx, user.ID)
	}
	if err != nil {
		return err
	}

	meilisearch.CommunityUpdateOrCreateDocumentIfEnabled(r.ctx, s.config, community)

	community.ViewerJoined = msql.NewNullBool(!req.Leave)
	community.ViewerMod = msql.NewNullBool(false)

	return w.writeJSON(community)
}

// @Summary		Get community mods.
// @Description	Get community mods.
// @Router			/api/communities/{communityID}/mods [GET]
// @Success		200
// @Tags			Community
// @Param			communityID	path	string	true	"Community ID"
func (s *Server) getCommunityMods(w *responseWriter, r *request) error {
	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, nil)
	if err != nil {
		return err
	}

	mods, err := core.GetCommunityMods(r.ctx, s.db, comm.ID)
	if err != nil {
		if httperr.IsNotFound(err) {
			return w.writeString("[]")
		}
		return err
	}

	return w.writeJSON(mods)
}

// @Summary		Add a community mod.
// @Description	Add a community mod.
// @Router			/api/communities/{communityID}/mods [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) addCommunityMod(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	username, ok := values["username"]
	if !ok {
		return httperr.NewBadRequest("empty_username", "Empty username.")
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}

	if err = core.MakeUserMod(r.ctx, s.db, comm, *r.viewer, user.ID, true); err != nil {
		return err
	}

	mods, err := core.GetCommunityMods(r.ctx, s.db, comm.ID)
	if err != nil {
		if httperr.IsNotFound(err) {
			return w.writeString("[]")
		}
		return err
	}

	return w.writeJSON(mods)
}

// @Summary		Remove a community mod.
// @Description	Remove a community mod.
// @Router			/api/communities/{communityID}/mods/{mod} [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
// @Param			mod				path	string	true	"Mod username"
func (s *Server) removeCommunityMod(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	vars := mux.Vars(r.req)
	cid, err := strToID(vars["communityID"])
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	username, ok := vars["mod"]
	if !ok {
		return httperr.NewBadRequest("empty_username", "Empty username.")
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}

	if err = core.MakeUserMod(r.ctx, s.db, comm, *r.viewer, user.ID, false); err != nil {
		return err
	}

	return w.writeJSON(user)
}

// @Summary		Get community rules.
// @Description	Get community rules.
// @Router			/api/communities/{communityID}/rules [GET]
// @Success		200
// @Tags			Community
// @Param			communityID	path	string	true	"Community ID"
func (s *Server) getCommunityRules(w *responseWriter, r *request) error {
	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}
	if err = comm.FetchRules(r.ctx); err != nil {
		return err
	}

	if comm.Rules == nil {
		return w.writeString("[]")
	}
	return w.writeJSON(comm.Rules)
}

// @Summary		Add a community rule.
// @Description	Add a community rule.
// @Router			/api/communities/{communityID}/rules [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) addCommunityRule(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	rule := core.CommunityRule{}
	if err := r.unmarshalJSONBody(&rule); err != nil {
		return err
	}

	if err = comm.AddRule(r.ctx, rule.Rule, rule.Description.String, *r.viewer); err != nil {
		return err
	}

	if err = comm.FetchRules(r.ctx); err != nil {
		return err
	}

	if comm.Rules == nil {
		return w.writeString("[]")
	}

	return w.writeJSON(comm.Rules)
}

// @Summary		Get a community rule.
// @Description	Get a community rule.
// @Router			/api/communities/{communityID}/rules/{ruleID} [GET]
// @Success		200
// @Tags			Community
// @Param			communityID	path	string	true	"Community ID"
// @Param			ruleID		path	string	true	"Rule ID"
func (s *Server) getCommunityRule(w *responseWriter, r *request) error {
	ruleID, err := strconv.Atoi(r.muxVar("ruleID"))
	if err != nil {
		return httperr.NewNotFound("rule_not_found", "Rule not found.")
	}

	rule, err := core.GetCommunityRule(r.ctx, s.db, uint(ruleID))
	if err != nil {
		return err
	}

	return w.writeJSON(rule)
}

// @Summary		Update a community rule.
// @Description	Update a community rule.
// @Router			/api/communities/{communityID}/rules/{ruleID} [PUT]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
// @Param			ruleID			path	string	true	"Rule ID"
func (s *Server) updateCommunityRule(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	ruleID, err := strconv.Atoi(r.muxVar("ruleID"))
	if err != nil {
		return httperr.NewNotFound("rule_not_found", "Rule not found.")
	}

	rule, err := core.GetCommunityRule(r.ctx, s.db, uint(ruleID))
	if err != nil {
		return err
	}

	req := core.CommunityRule{}
	if err := r.unmarshalJSONBody(&req); err != nil {
		return err
	}
	rule.Rule = req.Rule
	rule.Description = req.Description
	rule.ZIndex = req.ZIndex

	if err = rule.Update(r.ctx, *r.viewer); err != nil {
		return err
	}

	return w.writeJSON(rule)
}

// @Summary		Delete a community rule.
// @Description	Delete a community rule.
// @Router			/api/communities/{communityID}/rules/{ruleID} [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
// @Param			ruleID			path	string	true	"Rule ID"
func (s *Server) deleteCommunityRule(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	ruleID, err := strconv.Atoi(r.muxVar("ruleID"))
	if err != nil {
		return httperr.NewNotFound("rule_not_found", "Rule not found.")
	}

	rule, err := core.GetCommunityRule(r.ctx, s.db, uint(ruleID))
	if err != nil {
		return err
	}

	if err = rule.Delete(r.ctx, *r.viewer); err != nil {
		return err
	}

	return w.writeJSON(rule)
}

// @Summary		Report a post or comment.
// @Description	Report a post or comment.
// @Router			/api/_report [POST]
// @Success		200
// @Tags			Report
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) report(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	if err := s.rateLimit(r, "reporting_1_"+r.viewer.String(), time.Second*5, 1); err != nil {
		return err
	}
	if err := s.rateLimit(r, "reporting_2_"+r.viewer.String(), time.Hour*24, 50); err != nil {
		return err
	}

	inc := struct {
		Type     core.ReportType `json:"type"`
		TargetID uid.ID          `json:"targetId"`
		Reason   int             `json:"reason"`
	}{}
	if err := r.unmarshalJSONBody(&inc); err != nil {
		return err
	}

	var report *core.Report
	var err error
	if inc.Type == core.ReportTypePost {
		report, err = core.NewPostReport(r.ctx, s.db, inc.TargetID, inc.Reason, *r.viewer)
	} else if inc.Type == core.ReportTypeComment {
		report, err = core.NewCommentReport(r.ctx, s.db, inc.TargetID, inc.Reason, *r.viewer)
	} else {
		return httperr.NewBadRequest("invalid_report_type", "Invalid report type.")
	}
	if err != nil {
		return err
	}

	return w.writeJSON(report)
}

// @Summary		Get community reports.
// @Description	Get community reports.
// @Router			/api/communities/{communityID}/reports [GET]
// @Success		200
// @Tags			Report
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) getCommunityReports(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	query := r.urlQueryParams()

	limit, err := getFeedLimit(query, s.config.PaginationLimit, s.config.PaginationLimitMax)
	if err != nil {
		return err
	}

	page := 1
	if spage := query.Get("page"); spage != "" {
		if page, err = strconv.Atoi(spage); err != nil {
			return httperr.NewBadRequest("invalid_page", "Invalid page.")
		}
	}

	var t core.ReportType
	filter := query.Get("filter")
	switch filter {
	case "posts":
		t = core.ReportTypePost
	case "comments":
		t = core.ReportTypeComment
	case "all", "":
		t = core.ReportTypeAll
	default:
		return errInvalidFeedFilter
	}

	response := struct {
		Details core.CommunityReportsDetails `json:"details"`
		Reports []*core.Report               `json:"reports"`
		Limit   int                          `json:"limit"`
		Page    int                          `json:"page"`
	}{Limit: limit, Page: page}

	response.Details, err = core.FetchReportsDetails(r.ctx, s.db, cid)
	if err != nil {
		return err
	}

	response.Reports, err = core.GetReports(r.ctx, s.db, cid, t, limit, page)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	return w.writeJSON(response)
}

// @Summary		Delete a report.
// @Description	Delete a report.
// @Router			/api/communities/{communityID}/reports/{reportID} [DELETE]
// @Success		200
// @Tags			Report
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
// @Param			reportID		path	string	true	"Report ID"
func (s *Server) deleteReport(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	vars := mux.Vars(r.req)
	cid, err := strToID(vars["communityID"])
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	reportID, err := strconv.Atoi(vars["reportID"])
	if err != nil {
		return httperr.NewBadRequest("invalid_report_id", "Invalid report ID.")
	}
	report, err := core.GetReport(r.ctx, s.db, reportID)
	if err != nil {
		return err
	}
	if err = report.FetchTarget(r.ctx); err != nil {
		return err
	}
	if err = report.Delete(r.ctx, *r.viewer); err != nil {
		return err
	}

	return w.writeJSON(report)
}

// @Summary		Get community banned users.
// @Description	Get community banned users.
// @Router			/api/communities/{communityID}/banned [GET]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityGetBannedUsers(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	users, err := comm.GetBannedUsers(r.ctx)
	if err != nil {
		return err
	}
	if users == nil {
		return w.writeString("[]")
	}
	return w.writeJSON(users)
}

// @Summary		Ban  a user.
// @Description	Ban a user.
// @Router			/api/communities/{communityID}/banned [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityBanUser(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	username, ok := values["username"]
	if !ok {
		return httperr.NewBadRequest("no_username", "No username.")
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}
	if isMod, err := comm.UserMod(r.ctx, user.ID); err != nil {
		return err
	} else if r.req.Method == "POST" && (isMod || user.Admin) {
		// Cannot ban mod or admin.
		return httperr.NewForbidden("not_admin_nor_mod", "Neither an admin nor a mod.")
	}

	var expires *time.Time
	if expiresText, ok := values["expires"]; ok {
		expires = new(time.Time)
		if err = expires.UnmarshalText([]byte(expiresText)); err != nil {
			return httperr.NewBadRequest("invalid_expires", "Invalid expires.")
		}
	}

	err = comm.BanUser(r.ctx, *r.viewer, user.ID, expires)
	if err != nil {
		if msql.IsErrDuplicateErr(err) {
			return &httperr.Error{
				HTTPStatus: http.StatusConflict,
				Code:       "conflict",
				Message:    "There's a duplicate row.",
			}
		}
		return err
	}
	return w.writeJSON(user)
}

// @Summary		Unban a user.
// @Description	Unban a user.
// @Router			/api/communities/{communityID}/banned [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityUnbanUser(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	values, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	username, ok := values["username"]
	if !ok {
		return httperr.NewBadRequest("no_username", "No username.")
	}

	user, err := core.GetUserByUsername(r.ctx, s.db, username, nil)
	if err != nil {
		return err
	}

	var expires *time.Time
	if expiresText, ok := values["expires"]; ok {
		expires = new(time.Time)
		if err = expires.UnmarshalText([]byte(expiresText)); err != nil {
			return httperr.NewBadRequest("invalid_expires", "Invalid expires.")
		}
	}

	// Unban user.
	err = comm.UnbanUser(r.ctx, *r.viewer, user.ID)
	if err != nil {
		if msql.IsErrDuplicateErr(err) {
			return &httperr.Error{
				HTTPStatus: http.StatusConflict,
				Code:       "conflict",
				Message:    "There's a duplicate row.",
			}
		}
		return err
	}
	return w.writeJSON(user)
}

// @Summary		Upload a community profile picture.
// @Description	Upload a community profile picture.
// @Router			/api/communities/{communityID}/pro_pic [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityUploadProPic(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	r.req.Body = http.MaxBytesReader(w, r.req.Body, int64(s.config.MaxImageSize)) // limit max upload size
	if err := r.req.ParseMultipartForm(int64(s.config.MaxImageSize)); err != nil {
		return httperr.NewBadRequest("file_size_exceeded", "Max file size exceeded.")
	}

	file, _, err := r.req.FormFile("image")
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if err = comm.UpdateProPic(r.ctx, buf); err != nil {
		return err
	}

	return w.writeJSON(comm)
}

// @Summary		Delete a community profile picture.
// @Description	Delete a community profile picture.
// @Router			/api/communities/{communityID}/pro_pic [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityDeleteProPic(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	if err = comm.DeleteProPic(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(comm)
}

// @Summary		Upload a community banner image.
// @Description	Upload a community banner image.
// @Router			/api/communities/{communityID}/banner_image [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityUploadBannerImage(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	r.req.Body = http.MaxBytesReader(w, r.req.Body, int64(s.config.MaxImageSize)) // limit max upload size
	if err := r.req.ParseMultipartForm(int64(s.config.MaxImageSize)); err != nil {
		return httperr.NewBadRequest("file_size_exceeded", "Max file size exceeded.")
	}

	file, _, err := r.req.FormFile("image")
	if err != nil {
		return err
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if err = comm.UpdateBannerImage(r.ctx, buf); err != nil {
		return err
	}

	return w.writeJSON(comm)
}

// @Summary		Delete a community banner image.
// @Description	Delete a community banner image.
// @Router			/api/communities/{communityID}/banner_image [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			communityID		path	string	true	"Community ID"
func (s *Server) CommunityDeleteBannerImage(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	cid, err := strToID(r.muxVar("communityID"))
	if err != nil {
		return err
	}

	comm, err := core.GetCommunityByID(r.ctx, s.db, cid, r.viewer)
	if err != nil {
		return err
	}

	// Only mods and admins have access.
	if ok, err := userModOrAdmin(r.ctx, s.db, *r.viewer, comm); err != nil {
		return err
	} else if !ok {
		return errNotAdminNorMod
	}

	if err = comm.DeleteBannerImage(r.ctx); err != nil {
		return err
	}

	return w.writeJSON(comm)
}

// @Summary		Get community requests.
// @Description	Get community requests.
// @Router			/api/community_requests [GET]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"				default(Bearer <personal access token>)
// @Param			set				query	string	false	"Either 'all' or 'default' or 'subscribed'."	Enums(all,default,subscribed)
// @Param			sort			query	string	false	"Sort by 'size' or 'name'."						Enums(size,name)
func (s *Server) CommunityGetRequests(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if r.req.Method == "GET" {
		if !user.Admin {
			return httperr.NewForbidden("", "Not admin.")
		}

		items, err := core.GetCommunityRequests(r.ctx, s.db)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return w.writeString("[]")
		}
		return w.writeJSON(items)
	} else { // r.Method == "POST"

		if err := s.rateLimit(r, "req_comm_1_"+r.viewer.String(), time.Hour*12, 5); err != nil {
			return err
		}

		body, err := r.unmarshalJSONBodyToStringsMap(true)
		if err != nil {
			return err
		}

		note := body["note"]
		if len(note) > 2048 {
			note = note[:2048]
		}

		if err := core.CreateCommunityRequest(r.ctx, s.db, user.Username, body["name"], note); err != nil {
			return err
		}
		w.WriteHeader(http.StatusOK)
		return nil
	}
}

// @Summary		Create a community request.
// @Description	Create a community request.
// @Router			/api/community_requests [POST]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
func (s *Server) CommunityCreateRequests(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if err := s.rateLimit(r, "req_comm_1_"+r.viewer.String(), time.Hour*12, 5); err != nil {
		return err
	}

	body, err := r.unmarshalJSONBodyToStringsMap(true)
	if err != nil {
		return err
	}

	note := body["note"]
	if len(note) > 2048 {
		note = note[:2048]
	}

	if err := core.CreateCommunityRequest(r.ctx, s.db, user.Username, body["name"], note); err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// @Summary		Delete a community request.
// @Description	Delete a community request.
// @Router			/api/community_requests/{requestID} [DELETE]
// @Success		200
// @Tags			Community
// @Param			Authorization	header	string	true	"Insert your personal access token"	default(Bearer <personal access token>)
// @Param			requestID		path	string	true	"Request ID"
func (s *Server) deleteCommunityRequest(w *responseWriter, r *request) error {
	if !r.loggedIn {
		return errNotLoggedIn
	}

	user, err := core.GetUser(r.ctx, s.db, *r.viewer, nil)
	if err != nil {
		return err
	}

	if !user.Admin {
		return httperr.NewForbidden("", "Not admin.")
	}

	id, err := strconv.Atoi(r.muxVar("requestID"))
	if err != nil {
		return httperr.NewBadRequest("invalid_id", "Invalid request ID.")
	}

	if err := core.DeleteCommunityRequest(r.ctx, s.db, id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
