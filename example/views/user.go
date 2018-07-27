package views

import (
	"net/http"

	"github.com/MixinNetwork/ocean.one/example/models"
)

type UserView struct {
	UserId    string `json:"user_id"`
	SessionId string `json:"session_id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	MixinId   string `json:"mixin_id"`
}

func RenderUserWithAuthentication(w http.ResponseWriter, r *http.Request, user *models.User) {
	RenderDataResponse(w, r, UserView{
		UserId:    user.UserId,
		SessionId: user.SessionId,
		FullName:  user.FullName,
		Email:     user.Email.StringVal,
		Phone:     user.Phone.StringVal,
		MixinId:   user.MixinId.StringVal,
	})
}
