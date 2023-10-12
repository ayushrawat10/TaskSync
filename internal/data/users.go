package data

import (
	"context"
    "crypto/sha256"
	"errors"
	"fmt"
	"tasksync/internal/validator"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"created_at"`
	Name      string             `json:"name" bson:"name"`
	Email     string             `json:"email" bson:"email"`
	Password  []byte             `json:"-" bson:"password"`
	Activated bool               `json:"activated" bson:"activated"`
	Version   int32              `json:"version" bson:"version"`
}


func (user *User) SetPassword(plaintext string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}
	user.Password = hashedBytes
	return nil
}

func (user *User) PasswordMatches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(user.Password, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")
	ValidateEmail(v, user.Email)
	if user.Password == nil {
		panic("missing password hash for user")
	}
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
		if we, ok := err.(mongo.WriteException); ok {
			for _, e := range we.WriteErrors {
				if e.Code == 11000 {
					return ErrDuplicateEmail
                }
			}
		}
		return err
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return fmt.Errorf("could not convert to ObjectID")
	}
	user.ID = oid
	return nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"email": email}
	err := m.DB.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) GetByID(id primitive.ObjectID) (*User, error) {
    var user User

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{"_id": id}
    err := m.DB.FindOne(ctx, filter).Decode(&user)
    if err != nil {
        switch {
        case errors.Is(err, mongo.ErrNoDocuments):
            return nil, ErrRecordNotFound
        default:
            return nil, err
        }
    }
    return &user, nil
}

func (m UserModel) Update(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id": user.ID,
	}

	update := bson.M{
		"$set": bson.M{
			"name":     user.Name,
			"password": user.Password,
			"version":  user.Version + 1,
            "activated": user.Activated,
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	result := m.DB.FindOneAndUpdate(ctx, filter, update, opts)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return fmt.Errorf("edit conflict")
		}
		return result.Err()
	}

	var updatedUser User
	err := result.Decode(&updatedUser)
	if err != nil {
		return err
	}

	return nil
}

func (m UserModel) Delete(id int64) error {
	return nil
}

func (m TokenModel) GetForToken(tokenScope, tokenPlaintext string) (primitive.ObjectID, error) {
    tokenHash := sha256.Sum256([]byte(tokenPlaintext))

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    filter := bson.M{
        "hashedToken": tokenHash[:],
        "scope": tokenScope,
        "expiry": bson.M{"$gt": time.Now()},
    }

    var token Token
    err := m.DB.FindOne(ctx, filter).Decode(&token)
    if err != nil {
        switch {
        case errors.Is(err, mongo.ErrNoDocuments):
            return primitive.NilObjectID, ErrRecordNotFound
        default:
            return primitive.NilObjectID, err
        }
    }

    return token.UserID, nil
}
