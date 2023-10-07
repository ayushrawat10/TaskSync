package data

import (
    "go.mongodb.org/mongo-driver/mongo"
    "errors"
)

var (
    ErrRecordNotFound = errors.New("record not found")
    ErrEditConflict = errors.New("edit conflict")
)

type Models struct {
    Users UserModel
    Tokens TokenModel
}

func NewModels(db *mongo.Database) Models {
    return Models{
        Users: UserModel{DB: db.Collection("users")},
        Tokens: TokenModel{DB: db.Collection("tokens")},
    }
}
