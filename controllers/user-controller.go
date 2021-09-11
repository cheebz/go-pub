package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cheebz/go-pub/config"
	"github.com/cheebz/go-pub/models"
	"github.com/cheebz/go-pub/repositories"
	"github.com/cheebz/go-pub/responses"
	"github.com/gorilla/mux"
)

type controller struct{}

var (
	conf          config.Configuration
	repo          repositories.Repository
	accept        = "application/activity+json"
	acceptHeaders = http.Header{
		"Accept": []string{
			"application/activity+json",
			"application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"",
		},
	}
	contentType        = "application/activity+json"
	contentTypeHeaders = http.Header{
		"Content-Type": []string{
			"application/activity+json",
			"application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"",
		},
	}
)

func NewUserController(_conf config.Configuration, _repo repositories.Repository) UserController {
	conf = _conf
	repo = _repo
	return &controller{}
}

func (*controller) GetUser(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := repo.QueryUserByName(name)
	if err != nil {
		responses.NotFound(w, err)
		return
	}

	actor := generateActor(user.Name)
	w.Header().Set("Content-Type", contentType)
	json.NewEncoder(w).Encode(actor)
}

func generateActor(name string) models.Actor {
	return models.Actor{
		Object: models.Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
				map[string]interface{}{
					"manuallyApprovesFollowers": "as:manuallyApprovesFollowers",
				},
			},
			Id:      fmt.Sprintf("%s://%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name),
			Type:    "Person",
			Name:    name,
			Url:     fmt.Sprintf("%s://%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name),
			Summary: fmt.Sprintf("Summary of %s to come...", name), // TODO: Implement this
		},
		Inbox:                     fmt.Sprintf("%s://%s/%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name, conf.Endpoints.Inbox),
		Outbox:                    fmt.Sprintf("%s://%s/%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name, conf.Endpoints.Outbox),
		Following:                 fmt.Sprintf("%s://%s/%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name, conf.Endpoints.Following),
		Followers:                 fmt.Sprintf("%s://%s/%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name, conf.Endpoints.Followers),
		Liked:                     fmt.Sprintf("%s://%s/%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name, conf.Endpoints.Liked),
		PreferredUsername:         name,
		ManuallyApprovesFollowers: false, // TODO: Implement this
		PublicKey: models.PublicKey{
			ID:           fmt.Sprintf("%s://%s/%s/%s#main-key", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name),
			Owner:        fmt.Sprintf("%s://%s/%s/%s", conf.Protocol, conf.ServerName, conf.Endpoints.Users, name),
			PublicKeyPem: conf.RSAPublicKey,
		},
	}
}
