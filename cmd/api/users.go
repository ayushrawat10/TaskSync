package main

import (
	"fmt"
	"net/http"
    "time"
    "errors"
	"tasksync/internal/data"
	"tasksync/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
        ConfirmPassword string `json:"confirmPassword"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
    if input.Password != input.ConfirmPassword {
        v := validator.New()
        v.AddError("confirmPassword", "must match the password field")
        app.failedValidationResponse(w, r, v.Errors)
        return
    }

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.SetPassword(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

    token, err := app.models.Tokens.New(user.ID, 24 * time.Hour, data.ScopeActivation)
    if err != nil {
        app.serverErrorResponse(w, r, err)
        return
    }

    app.background(func() {
        data := map[string]interface{} {
            "activationToken": token.PlainToken,
        }

        err = app.mailer.Send(user.Email, "user_welcome.tmpl.html", data)
        if err != nil {
            app.logger.PrintError(err, nil)
        }
    })

	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showUserHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Show the details of a specific user...")
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
    var input struct {
        TokenPlaintext string `json:"token"`
    }

    err := app.readJSON(w, r, &input)
    if err != nil {
        app.badRequestResponse(w, r, err)
        return
    }

    v := validator.New()

    if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
        app.failedValidationResponse(w, r, v.Errors)
        return
    }

    userID, err := app.models.Tokens.GetForToken(data.ScopeActivation, input.TokenPlaintext)
    if err != nil {
        switch {
        case errors.Is(err, data.ErrRecordNotFound):
            v.AddError("token", "invalid or expired activation token")
            app.failedValidationResponse(w, r, v.Errors)
        default:
            app.serverErrorResponse(w, r, err)
        }
        return
    }
    user, err := app.models.Users.GetByID(userID)
    if err != nil {
        switch {
        case errors.Is(err, data.ErrRecordNotFound):
            app.notFoundResponse(w, r)
        default:
            app.serverErrorResponse(w, r, err)
        }
        return
    }

    user.Activated = true
    err = app.models.Users.Update(user)
    if err != nil {
        switch {
        // case errors.Is(err, data.ErrEditConflict):
        //     app.editConflictResponse(w, r)
        default:
            app.serverErrorResponse(w, r, err)
        }
        return
    }

        err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
        if err != nil {
            app.serverErrorResponse(w, r, err)
            return
        }

        err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
        if err != nil {
            app.serverErrorResponse(w, r, err)
        }
}
