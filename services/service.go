package services

import (
	"github.com/cheebz/arb"
	"github.com/cheebz/go-pub/models"
)

type Service interface {
	DiscoverUserByName(name string) (models.User, error)
	GetUserByName(name string) (models.User, error)
	CheckUser(name string) error
	CreateUser(name string) (string, error)
	GetInboxTotalItemsByUserName(name string) (int, error)
	GetInboxByUserName(name string) ([]models.Activity, error)
	GetOutboxTotalItemsByUserName(name string) (int, error)
	GetOutboxByUserName(name string) ([]models.Activity, error)
	GetFollowersTotalItemsByUserName(name string) (int, error)
	GetFollowersByUserName(name string) ([]string, error)
	GetFollowingTotalItemsByUserName(name string) (int, error)
	GetFollowingByUserName(name string) ([]string, error)
	GetLikedTotalItemsByUserName(name string) (int, error)
	GetLikedByUserName(name string) ([]models.Object, error)
	GetActivity(ID int) (models.Activity, error)
	GetObject(ID int) (models.Object, error)
	SaveInboxActivity(activityArb arb.Arb, name string) (arb.Arb, error)
	SaveOutboxActivity(activityArb arb.Arb, name string) (arb.Arb, error)
}
