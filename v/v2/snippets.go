package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/cihub/seelog"
	httpr "github.com/julienschmidt/httprouter"
	"github.com/sniperkit/api/ctxext"
	"github.com/sniperkit/api/domain"
	"github.com/sniperkit/api/endpoints"
	"github.com/sniperkit/api/types"
	"github.com/sniperkit/api/v"
)

func matchOrganizationForUser(rawOwner string, userId string) (string, error) {
	userOrganizationDao := domain.NewUserOrganizationDao(db)
	orgz, err := userOrganizationDao.ListOrganizationsForUser(userId)
	if err != nil {
		return "", fmt.Errorf("database error: %s", err.Error())
	}
	if len(orgz) == 0 {
		return "", fmt.Errorf("user is part of no organizations")
	}
	organizationDao := domain.NewOrganizationDao(db)
	matches, err := organizationDao.MatchesInIds(orgz, rawOwner)
	if err != nil {
		return "", fmt.Errorf("database error: %s", err.Error())
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no organizations match the pattern: %s", rawOwner)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("pattern %s matches multiples organizations", rawOwner)
	}
	return matches[0].Name, nil
}

func getRealOwner(rawOwner string, userId string) (string, error) {
	var index string
	if rawOwner == "me" {
		// this will be user specific content
		index = userId
	} else if rawOwner == "" {
		// borg is the global index
		index = endpoints.PublicBorgSnippet
	} else {
		// here we consider this is organizastion specific stuff
		// let's try to match one user org with this string
		return matchOrganizationForUser(rawOwner, userId)
	}
	return index, nil
}

func q(w http.ResponseWriter, r *http.Request, p httpr.Params) {
	size := 5
	s, err := strconv.ParseInt(r.FormValue("l"), 10, 32)
	if err == nil && s > 0 {
		size = int(s)
	}
	res, err := ep.Query(r.FormValue("q"), size, r.FormValue("p") == "true")
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, err.Error())
	}
	bs, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, string(bs))
}

func getLatestSnippets(
	ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	owner := p.ByName("owner")
	if len(owner) == 0 {
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Missing owner url parameter")
		return
	}

	// this is a maybeAuth endpoint, so first check if the guys is auth
	// if not use "borg", else try to figure out if it is a private thing
	var index string
	if isAuth, _ := ctxext.IsAuth(ctx); isAuth {
		var err error
		userId, _ := ctxext.UserId(ctx)
		index, err = getRealOwner(owner, userId)
		if err != nil {
			common.WriteResponse(w, http.StatusBadRequest, fmt.Sprintf("borg-api: %s", err.Error()))
			return
		}
	} else {
		// by default if not auth index is the public one
		index = endpoints.PublicBorgSnippet
	}

	res, err := ep.GetLatestSnippets(index)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, err.Error())
	}
	bs, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	common.WriteResponse(w, http.StatusOK, string(bs))
}

func createSnippet(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}
	s := struct {
		Snippet types.Problem
		Owner   string
	}{}

	if err := json.Unmarshal(body, &s); err != nil {
		log.Errorf("[createSnippet] Invalid snippet, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid snippet")
		return
	}

	userId, _ := ctxext.UserId(ctx)
	index, err := getRealOwner(s.Owner, userId)
	if err != nil {
		common.WriteResponse(w, http.StatusBadRequest, fmt.Sprintf("borg-api: %s", err.Error()))
		return
	}
	err = ep.CreateSnippet(&s.Snippet, index, userId)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to unmarshal snippet")
		return
	}
	common.WriteJsonResponse(w, http.StatusOK, s.Snippet)
}

func updateSnippet(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}

	s := struct {
		Snippet types.Problem
		Owner   string
	}{}

	if err := json.Unmarshal(body, &s); err != nil {
		log.Errorf("[updateSnippet] Invalid snippet, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid snippet")
		return
	}

	userId, _ := ctxext.UserId(ctx)
	index, err := getRealOwner(s.Owner, userId)
	if err != nil {
		common.WriteResponse(w, http.StatusBadRequest, fmt.Sprintf("borg-api: %s", err.Error()))
		return
	}

	err = ep.UpdateSnippet(&s.Snippet, index, ctx.Value("userId").(string))
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: error")
		return
	}
	common.WriteResponse(w, http.StatusOK, "{}")
}

func snippetWorked(ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: unable to read body")
		return
	}
	s := struct {
		Query string
		Id    string
	}{}
	if err := json.Unmarshal(body, &s); err != nil {
		log.Errorf("[updateSnippet] invalid worked request, %s, input was %s", err.Error(), string(body))
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Invalid worked request")
		return
	}
	err = ep.Worked(s.Id, s.Query)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: error: "+err.Error())
		return
	}
	common.WriteResponse(w, http.StatusOK, "{}")
}

func getSnippet(
	ctx context.Context, w http.ResponseWriter, r *http.Request, p httpr.Params) {
	id := p.ByName("id")
	if len(id) == 0 {
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Missing id url parameter")
		return
	}
	owner := p.ByName("owner")
	if len(owner) == 0 {
		common.WriteResponse(w, http.StatusBadRequest, "borg-api: Missing owner url parameter")
		return
	}

	// this is a maybeAuth endpoint, so first check if the guys is auth
	// if not use "borg", else try to figure out if it is a private thing
	var index string
	if isAuth, _ := ctxext.IsAuth(ctx); isAuth {
		var err error
		userId, _ := ctxext.UserId(ctx)
		index, err = getRealOwner(owner, userId)
		if err != nil {
			common.WriteResponse(w, http.StatusBadRequest, fmt.Sprintf("borg-api: %s", err.Error()))
			return
		}
	} else {
		// by default if not auth index is the public one
		index = endpoints.PublicBorgSnippet
	}

	snipp, err := ep.GetSnippet(index, id)
	if err != nil {
		common.WriteResponse(w, http.StatusInternalServerError, "borg-api: Failed to get snippet")
		return
	}
	if snipp == nil {
		common.WriteResponse(w, http.StatusNotFound, "borg-api: snippet not found")
		return
	}
	bs, _ := json.Marshal(snipp)
	common.WriteResponse(w, http.StatusOK, string(bs))
}
