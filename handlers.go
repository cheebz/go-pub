package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func sinkHandler(w http.ResponseWriter, r *http.Request) {
	notFound(w, errors.New("endpoint does not exist"))
	return
}

func getWebFinger(w http.ResponseWriter, r *http.Request) {
	resource := r.FormValue("resource")
	name, err := parseResource(resource)
	if err != nil {
		badRequest(w, err)
		return
	}

	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	if !user.Discoverable {
		notFound(w, errors.New("user is not discoverable"))
		return
	}

	webfinger := generateWebFinger(user.Name)
	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(webfinger)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	actor := generateActor(user.Name)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(actor)
}

func getInbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}
	claims, _ := checkJWTClaims(r)
	if claims.Username != name {
		unauthorizedRequest(w, errors.New("not your inbox"))
		return
	}

	page := r.FormValue("page")
	if page != "true" {
		totalItems, err := queryInboxTotalItemsByUserName(user.Name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		inbox := generateOrderedCollection(user.Name, config.Endpoints.Inbox, totalItems)
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(inbox)
		return
	}

	activities, err := queryInboxByUserName(user.Name)
	if err != nil {
		internalServerError(w, err)
		return
	}

	orderedItems := make([]interface{}, len(activities))
	for i, activity := range activities {
		orderedItems[i] = activity
	}

	inboxPage := generateOrderedCollectionPage(name, config.Endpoints.Inbox, orderedItems)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(inboxPage)
}

func postInbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	err := checkUser(name)
	if err != nil {
		badRequest(w, err)
		return
	}
	recipient := fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name)
	err = checkContentType(r.Header)
	if err != nil {
		badRequest(w, err)
		return
	}
	// TODO: Check signature here...
	activityArb, err := parsePayload(r)
	if err != nil {
		badRequest(w, err)
		return
	}
	activityIRI, err := activityArb.GetString("id")
	if err != nil {
		badRequest(w, err)
		return
	}
	actorArb, err := findProp(activityArb, "actor", acceptHeaders)
	if err != nil {
		badRequest(w, err)
		return
	}
	actorIRI, err := actorArb.GetString("id")
	if err != nil {
		badRequest(w, err)
		return
	}
	objectArb, err := findProp(activityArb, "object", acceptHeaders)
	if err != nil {
		badRequest(w, err)
		return
	}
	objectIRI, err := objectArb.GetString("id")
	if err != nil {
		badRequest(w, err)
		return
	}
	activityType, err := activityArb.GetString("type")
	if err != nil {
		badRequest(w, err)
		return
	}
	switch activityType {
	case "Create":
		// TODO: Save more of the object, if possible
		_, err = createInboxActivity(activityArb, objectArb, actorIRI, recipient)
		if err != nil {
			internalServerError(w, err)
			return
		}
	case "Follow":
		if objectIRI != recipient {
			badRequest(w, errors.New("wrong inbox"))
			return
		}
		_, err = createInboxReferenceActivity(activityArb, recipient, actorIRI, recipient)
		if err != nil {
			internalServerError(w, err)
			return
		}
		responseArb, err := newActivityArbReference(activityIRI, "Accept")
		if err != nil {
			internalServerError(w, err)
			return
		}
		responseArb["actor"] = recipient
		responseArb, err = createOutboxReferenceActivity(responseArb)
		if err != nil {
			internalServerError(w, err)
			return
		}
		fedChan <- Federation{Name: name, Recipient: actorIRI, Data: responseArb.ToBytes()}
	default:
		badRequest(w, errors.New("unsupported activity type"))
		return
	}

	accepted(w)
}

func getOutbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	page := r.FormValue("page")
	if page != "true" {
		totalItems, err := queryOutboxTotalItemsByUserName(user.Name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		outbox := generateOrderedCollection(user.Name, config.Endpoints.Outbox, totalItems)
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(outbox)
		return
	}

	activities, err := queryOutboxByUserName(user.Name)
	if err != nil {
		internalServerError(w, err)
		return
	}

	orderedItems := make([]interface{}, len(activities))
	for i, activity := range activities {
		orderedItems[i] = activity
	}

	outboxPage := generateOrderedCollectionPage(name, config.Endpoints.Outbox, orderedItems)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(outboxPage)
}

func postOutbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	claims, _ := checkJWTClaims(r)
	if claims.Username != name {
		unauthorizedRequest(w, errors.New("not your outbox"))
		return
	}
	actor := fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, claims.Username)
	err := checkContentType(r.Header)
	if err != nil {
		badRequest(w, err)
		return
	}
	activityArb, err := parsePayload(r)
	if err != nil {
		badRequest(w, err)
		return
	}
	activityType, err := activityArb.GetString("type")
	if err != nil {
		badRequest(w, err)
		return
	}
	activityArb["actor"] = actor
	switch activityType {
	case "Create":
		objectArb, err := activityArb.GetArb("object")
		if err != nil {
			badRequest(w, err)
			return
		}
		objectArb["attributedTo"] = actor
		activityArb, err = createOutboxActivityDetail(activityArb, objectArb)
		if err != nil {
			internalServerError(w, err)
			return
		}
	case "Like":
		activityArb, err = createOutboxReferenceActivity(activityArb)
		if err != nil {
			internalServerError(w, err)
			return
		}
	default:
		badRequest(w, errors.New("unsupported activity type"))
		return
		// Activity type is something else, save object reference (if new), Activity, and Activity_to
	}

	// Get recipients
	recipients, err := getRecipients(activityArb, "to")
	if err != nil {
		log.Println(err.Error())
	}
	// Deliver to recipients
	for _, recipient := range recipients {
		// TODO: Handle individual user vs. followers/following collections
		if recipient.Host != config.ServerName {
			fedChan <- Federation{Name: name, Recipient: recipient.String(), Data: activityArb.ToBytes()}
		}
		if strings.HasSuffix(recipient.String(), "/followers") {
			fedChan <- Federation{Name: name, Recipient: recipient.String(), Data: activityArb.ToBytes()}
		}
		err = addActivityTo(activityArb, recipient.String())
		if err != nil {
			log.Println(err.Error())
		}
	}

	w.Header().Set("Content-Type", contentType)
	iri, err := activityArb.GetString("id")
	created(w, iri)
	activityArb.Write(w)
}

func getFollowing(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	page := r.FormValue("page")
	if page != "true" {
		totalItems, err := queryFollowingTotalItemsByUserName(user.Name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		following := generateOrderedCollection(user.Name, config.Endpoints.Following, totalItems)
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(following)
		return
	}

	following, err := queryFollowingByUserName(user.Name)
	if err != nil {
		internalServerError(w, err)
		return
	}

	orderedItems := make([]interface{}, len(following))
	for i, actor := range following {
		orderedItems[i] = actor
	}

	followingPage := generateOrderedCollectionPage(user.Name, config.Endpoints.Following, orderedItems)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(followingPage)
}

func getFollowers(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	page := r.FormValue("page")
	if page != "true" {
		totalItems, err := queryFollowersTotalItemsByUserName(user.Name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		followers := generateOrderedCollection(user.Name, config.Endpoints.Followers, totalItems)
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(followers)
		return
	}

	followers, err := queryFollowersByUserName(user.Name)
	if err != nil {
		internalServerError(w, err)
		return
	}

	orderedItems := make([]interface{}, len(followers))
	for i, actor := range followers {
		orderedItems[i] = actor
	}

	followersPage := generateOrderedCollectionPage(user.Name, config.Endpoints.Followers, orderedItems)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(followersPage)
}

func getLiked(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := queryUserByName(name)
	if err != nil {
		notFound(w, err)
		return
	}

	page := r.FormValue("page")
	if page != "true" {
		totalItems, err := queryLikedTotalItemsByUserName(user.Name)
		if err != nil {
			internalServerError(w, err)
			return
		}

		liked := generateOrderedCollection(user.Name, config.Endpoints.Liked, totalItems)
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(liked)
		return
	}

	liked, err := queryLikedByUserName(user.Name)
	if err != nil {
		internalServerError(w, err)
		return
	}

	orderedItems := make([]interface{}, len(liked))
	for i, activity := range liked {
		orderedItems[i] = activity
	}

	likedPage := generateOrderedCollectionPage(user.Name, config.Endpoints.Liked, orderedItems)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(likedPage)
}

func getActivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		badRequest(w, err)
		return
	}

	activity, err := queryActivity(id)
	if err != nil {
		notFound(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(activity)
}

func getObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		badRequest(w, err)
		return
	}

	object, err := queryObject(id)
	if err != nil {
		notFound(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(object)
}
