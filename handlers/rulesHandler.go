package handlers

import (
	"net/http"

	"github.com/NamedKitten/kittehbooru/i18n"
	templates "github.com/NamedKitten/kittehbooru/template"
)

type RulesTemplateData struct {
	RulesLineCount int
	templates.T
}

func RulesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !DB.SetupCompleted {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}
	user, loggedIn := DB.CheckForLoggedInUser(ctx, r)
	x := RulesTemplateData{
		len(DB.Settings.Rules),
		templates.T{
			LoggedIn:     loggedIn,
			LoggedInUser: user,
			Translator:   i18n.GetTranslator(r),
		},
	}
	err := templates.RenderTemplate(w, "rules.html", x)
	if err != nil {
		renderError(w, "TEMPLATE_RENDER_ERROR", err, http.StatusBadRequest)
	}
}
