package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"tasksync/internal/validator"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	PlainToken  string             `json:"token" bson:"-"`
	HashedToken []byte             `json:"-" bson:"hashedToken"`
	UserID      primitive.ObjectID `json:"-" bson:"userID"`
	Expiry      time.Time          `json:"expiry" bson:"expiry"`
	Scope       string             `json:"-" bson:"scope"`
}

func generateToken(userID primitive.ObjectID, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Scope:  scope,
		Expiry: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.PlainToken = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.PlainToken))
	token.HashedToken = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *mongo.Collection
}

func (m TokenModel) New(userID primitive.ObjectID, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, nil
}

func (m TokenModel) Insert(token *Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.DB.InsertOne(ctx, token)
	return err
}

func (m TokenModel) GetUserIDForToken(tokenScope, tokenPlaintext string) (primitive.ObjectID, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"hashedToken": tokenHash[:],
		"scope":       tokenScope,
		"expiry":      bson.M{"$gt": time.Now()},
	}

	var token Token
	err := m.DB.FindOne(ctx, filter).Decode(&token)
	if err != nil {
		switch {
		case err == mongo.ErrNoDocuments:
			return primitive.NilObjectID, ErrRecordNotFound
		default:
			return primitive.NilObjectID, err
		}
	}
	return token.UserID, nil
}

func (m TokenModel) DeleteAllForUser(scope string, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.DB.DeleteMany(ctx, bson.M{
		"scope":  scope,
		"userID": userID,
	})
	return err
}
