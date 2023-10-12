package main

import (
    "errors"
    "net/http"
    "time"
    "tasksync/internal/data"
    "tasksync/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Email string `json:"email" bson:"email"`
        Password string `json:"password"`
    }

    err := app.readJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    v := validator.New()
    data.ValidateEmail(v, input.Email)
    data.ValidatePasswordPlaintext(v, input.Password)
    if !v.Valid() {
        app.failedValidationResponse(w, r, v.Errors)
        return
    }

    user, err := app.models.Users.GetByEmail(input.Email)
    if err != nil {
        switch {
        case errors.Is(err, data.ErrRecordNotFound):
            app.invalidCredentialsResponse(w, r)
        default:
            app.serverErrorResponse(w, r, err)
        }
        return
    }

    match, err := user.PasswordMatches(input.Password)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    if !match {
        app.invalidCredentialsResponse(w, r)
        return
    }

    token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}
