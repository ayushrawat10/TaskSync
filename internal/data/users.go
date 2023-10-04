package data

import (
	"context"
	"tasksync/internal/validator"
	"time"
    "fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    CreatedAt time.Time `bson:"created_at"`
    Name string `json:"name" bson:"name"`
    Email string `json:"email" bson:"email"`
    Password string `json:"password" bson:"password"`
    Version int32 `json:"version" bson:"version"`
}

func ValidateUser(v *validator.Validator, user *User) {
    v.Check(user.Name != "", "name", "must be provided")
    v.Check(len(user.Name) <= 100, "name", "must not be more than 100 bytes long")
    v.Check(user.Email != "", "email", "must be provided")
    v.Check(len(user.Email) <= 100, "email", "must not be more than 100 bytes long")
    v.Check(user.Password != "", "password", "must be provided")
    v.Check(len(user.Password) >= 8, "password", "must be at least 8 bytes long")
    v.Check(len(user.Password) <= 72, "password", "must not be more than 72 bytes long")
    v.Check(user.Password != user.Name, "password", "must not be the same as name")
}

type UserModel struct {
    DB *mongo.Collection
}

func (m UserModel) Insert(user *User) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    user.CreatedAt = time.Now()
    user.Version = 1

    result, err := m.DB.InsertOne(ctx, user)
    if err != nil {
        return err
    }
    oid, ok := result.InsertedID.(primitive.ObjectID)
    if !ok {
        return fmt.Errorf("could not convert to ObjectID")
    }
    user.ID = oid
    return nil
}

func (m UserModel) Get(id int64) (*User, error) {
    return nil, nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {
    return nil, nil
}

func (m UserModel) Update(user *User) error {
    return nil
}

func (m UserModel) Delete(id int64) error {
    return nil
}
