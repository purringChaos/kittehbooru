package handlers

import (
	"net/http"

	"github.com/NamedKitten/kittehbooru/i18n"
	templates "github.com/NamedKitten/kittehbooru/template"
	"github.com/NamedKitten/kittehbooru/types"
	"github.com/gorilla/mux"
)

type UserResultsTemplate struct {
	AvatarPost   types.Post
	User         types.User
	IsAbleToEdit bool
	templates.T
}

func UserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !DB.SetupCompleted {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	vars := mux.Vars(r)
	loggedInUser, loggedIn := DB.CheckForLoggedInUser(ctx, r)

	username := vars["userID"]
	user, err := DB.User(ctx, vars["userID"])
	if err != nil {
		renderError(w, "USER_NOT_FOUND", err, http.StatusBadRequest)
		return
	}

	avatarPost, err := DB.Post(ctx, user.AvatarID)
	if err != nil {
		avatarPost = types.Post{}
	}
	templateInfo := UserResultsTemplate{
		AvatarPost:   avatarPost,
		User:         user,
		IsAbleToEdit: (loggedInUser.Username == username) && loggedIn,
		T: templates.T{
			LoggedIn:     loggedIn,
			LoggedInUser: loggedInUser,
			Translator:   i18n.GetTranslator(r),
		},
	}

	err = templates.RenderTemplate(w, "user.html", templateInfo)
	if err != nil {
		renderError(w, "TEMPLATE_RENDER_ERROR", err, http.StatusBadRequest)
	}
}
