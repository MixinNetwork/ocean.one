package views

import (
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/models"
)

type UserView struct {
	OceanToken string `json:"ocean_token"`
	MixinToken string `json:"mixin_token"`
}

func RenderUserWithAuthentication(w http.ResponseWriter, r *http.Request, user *models.User) {
}
