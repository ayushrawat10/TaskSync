package main

import (
	"fmt"
	"net/http"
    "tasksync/internal/data"
    "tasksync/internal/validator"
)

func (app *application) createUserHandler(w http.ResponseWriter, r *http.Request) {
    var input struct {
        Name string `json:"name"`
        Email string `json:"email"`
        Password string `json:"password"`
        ConfirmPassword string `json:"confirmPassword"`
    }
    err := app.readJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    user := &data.User{
        Name: input.Name,
        Email: input.Email,
        Password: input.Password,
    }

    v := validator.New()
    v.Check(input.ConfirmPassword != input.Password, "confirmPassword", "passwords must match")
    if data.ValidateUser(v, user); !v.Valid() {
        app.failedValidationResponse(w, r, v.Errors)
        return
    }

    err = app.models.Users.Insert(user)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
    if err != nil {
        app.serverErrorResponse(w, r, err)
    }
}

func (app *application) showUserHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Show the details of a specific user...")
}

