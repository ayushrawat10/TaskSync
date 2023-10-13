package data

import (
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users  UserModel
	Tokens TokenModel
}

func NewModels(db *mongo.Database) Models {
	return Models{
		Users:  UserModel{DB: db.Collection("users")},
		Tokens: TokenModel{DB: db.Collection("tokens")},
	}
}
