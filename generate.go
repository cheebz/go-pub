package main

import (
	"fmt"
)

// TODO: Move this stuff into a package

func generateWebFinger(name string) WebFinger {
	return WebFinger{
		Subject: fmt.Sprintf("acct:%s@%s", name, config.ServerName),
		Aliases: []string{
			fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
		},
		Links: append(
			[]WebFingerLink{},
			WebFingerLink{
				Rel:  "self",
				Type: "application/activity+json",
				Href: fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
			},
		),
	}
}

func generateActor(name string) Actor {
	return Actor{
		Object: Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
				map[string]interface{}{
					"manuallyApprovesFollowers": "as:manuallyApprovesFollowers",
				},
			},
			Id:      fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
			Type:    "Person",
			Name:    name,
			Url:     fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
			Summary: fmt.Sprintf("<p>Summary of %s to come...</p>", name), // TODO: Implement this
		},
		Inbox:     fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, config.Endpoints.Inbox),
		Outbox:    fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, config.Endpoints.Outbox),
		Following: fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, config.Endpoints.Following),
		Followers: fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, config.Endpoints.Followers),
		// Liked:                     fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, config.Endpoints.Liked),
		PreferredUsername:         name,
		ManuallyApprovesFollowers: false, // TODO: Implement this
		PublicKey: PublicKey{
			ID:           fmt.Sprintf("%s://%s/%s/%s#main-key", config.Protocol, config.ServerName, config.Endpoints.Users, name),
			Owner:        fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
			PublicKeyPem: config.RSAPublicKey,
		},
	}
}

func generateNewActivity() Activity {
	var activity Activity
	activity.Context = []interface{}{
		"https://www.w3.org/ns/activitystreams",
		"https://w3id.org/security/v1",
	}
	return activity
}

func generateOrderedCollection(name string, endpoint string, totalItems int) OrderedCollection {
	return OrderedCollection{
		Object: Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
			},
			Id:   fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, endpoint),
			Type: "OrderedCollection",
		},
		TotalItems: totalItems,
		First:      fmt.Sprintf("%s://%s/%s/%s/%s?page=true", config.Protocol, config.ServerName, config.Endpoints.Users, name, endpoint),
		Last:       fmt.Sprintf("%s://%s/%s/%s/%s?min_id=0&page=true", config.Protocol, config.ServerName, config.Endpoints.Users, name, endpoint),
	}
}

func generateOrderedCollectionPage(name string, endpoint string, orderedItems []interface{}) OrderedCollectionPage {
	return OrderedCollectionPage{
		Object: Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
			},
			Id:   fmt.Sprintf("%s://%s/%s/%s/%s?page=true", config.Protocol, config.ServerName, config.Endpoints.Users, name, endpoint),
			Type: "OrderedCollectionPage",
		},
		PartOf:       fmt.Sprintf("%s://%s/%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name, endpoint),
		OrderedItems: orderedItems,
	}
}

func generatePostActivity(post Note) PostActivityResource {
	// TODO: Get this array
	to := []string{
		"https://www.w3.org/ns/activitystreams#Public",
	}
	for _, url := range post.Activity.To {
		to = append(to, url)
	}

	return PostActivityResource{
		Object: Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
			},
			Type: post.Activity.Type,
			Id:   fmt.Sprintf("%s://%s/%s/%s/activities/%d", config.Protocol, config.ServerName, config.Endpoints.Users, post.Activity.UserName, post.Activity.ID),
			To:   to,
		},
		Actor: fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, post.Activity.UserName),
		ChildObject: Object{
			Context: []interface{}{
				"https://www.w3.org/ns/activitystreams",
				"https://w3id.org/security/v1",
			},
			Type:    "Note",
			Id:      fmt.Sprintf("%s://%s/%s/%s/posts/%d", config.Protocol, config.ServerName, config.Endpoints.Users, post.UserName, post.ID),
			Name:    fmt.Sprintf("A note from %s", post.UserName),
			Content: post.Content,
		},
	}
}
