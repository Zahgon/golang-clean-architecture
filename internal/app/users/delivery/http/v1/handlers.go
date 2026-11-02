package v1

import (
	"github.com/DoWithLogic/golang-clean-architecture/internal/app/users"
	"github.com/DoWithLogic/golang-clean-architecture/internal/app/users/dtos"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/binder"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/observability/instrumentation"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/response"
	"github.com/gin-gonic/gin"
)

type handlers struct {
	uc users.Usecase
}

func NewHandlers(uc users.Usecase) *handlers {
	return &handlers{uc}
}

// @Summary		Login
// @Description	Login
// @ID			login
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		body	body		dtos.UserLoginRequest							true	"Login Request"
// @Success		200  	{object}	response.Success{data=dtos.UserLoginResponse}			"SUCCESS"
// @Failure		500		{object}	response.FailedResponse									"INTERNAL_SERVER__ERROR"
// @Router		/user/public/login [post]
func (h *handlers) LoginHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "LoginHandler")
	defer span.End()

	var request dtos.UserLoginRequest
	if err := binder.Bind(c, &request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	if err := request.Validate(); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	authData, err := h.uc.Login(ctx, request)
	if err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(authData).Send(c)
}

// @Summary		Sign Up
// @Description	Sign Up
// @ID			sign-up
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		body	body		dtos.SignUpRequest		true	"Sign Up Request"
// @Success		200		{object}	response.ResponseFormat			"SUCCESS"
// @Failure		500		{object}	response.FailedResponse			"INTERNAL_SERVER__ERROR"
// @Router		/user/public/sign-up [post]
func (h *handlers) SignUpHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "SignUpHandler")
	defer span.End()

	request := new(dtos.SignUpRequest)
	if err := binder.Bind(c, request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	if err := request.Validate(); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	if err := h.uc.SignUp(ctx, *request); err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(nil).Send(c)
}

// @Summary		User Detail By ID
// @Description	User Detail By ID
// @ID			user-detail-by-id
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		id		path		int									true	"User ID"
// @Success		200  	{object}	response.Success{data=dtos.User}			"SUCCESS"
// @Failure		500		{object}	response.FailedResponse						"INTERNAL_SERVER__ERROR"
// @Router		/user/{id}/detail [get]
// @Security	BearerToken
func (h *handlers) UserDetailByIDHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "UserDetailByIDHandler")
	defer span.End()

	request := new(dtos.UserDetailByIDRequest)
	if err := binder.Bind(c, request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	userData, err := h.uc.UserDetail(ctx, *request)
	if err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(userData).Send(c)
}

// @Summary		User Detail By Contact
// @Description	User Detail By Contact
// @ID			user-detail-by-contact
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		contact_value		path		string								true	"User Contact"
// @Success		200  				{object}	response.Success{data=dtos.User}			"SUCCESS"
// @Failure		500					{object}	response.FailedResponse						"INTERNAL_SERVER__ERROR"
// @Router		/user/contact/{contact_value}/detail [get]
// @Security	BearerToken
func (h *handlers) UserDetailByContactValueHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "UserDetailByContactValueHandler")
	defer span.End()

	request := new(dtos.UserDetailByContactValueRequest)
	if err := binder.Bind(c, request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	userData, err := h.uc.UserDetail(ctx, *request)
	if err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(userData).Send(c)
}

// @Summary		Update User
// @Description	Update User
// @ID			update-user
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		id		path		int										true	"User ID"
// @Param		body	body		dtos.UserUpdateRequest					true	"Update User Request"
// @Success		200		{object}	response.ResponseFormat							"SUCCESS"
// @Failure		500		{object}	response.FailedResponse							"INTERNAL_SERVER__ERROR"
// @Router		/user/{id}/update [patch]
// @Security	BearerToken
func (h *handlers) UpdateUserHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "UpdateUserHandler")
	defer span.End()

	request := new(dtos.UserUpdateRequest)
	if err := binder.Bind(c, request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	if err := h.uc.UserUpdate(ctx, *request); err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(nil).Send(c)
}

// @Summary		Transition User Status
// @Description	Transition User Status
// @ID			transition-user-status
// @Tags		Users
// @Accept		json
// @Produce		json
// @Param		id		path		int										true	"User ID"
// @Param		body	body		dtos.TransitionUserStatusRequest		true	"Transition User Status Request"
// @Success		200		{object}	response.ResponseFormat							"SUCCESS"
// @Failure		500		{object}	response.FailedResponse							"INTERNAL_SERVER__ERROR"
// @Router		/user/{id}/status/transition [put]
// @Security	BearerToken
func (h *handlers) TransitionUserStatusHandler(c *gin.Context) {
	ctx, span := instrumentation.NewTraceSpan(c.Request.Context(), "TransitionUserStatusHandler")
	defer span.End()

	request := new(dtos.TransitionUserStatusRequest)
	if err := binder.Bind(c, request); err != nil {
		response.ErrorBuilder(response.BadRequest(err)).Send(c)
		return
	}

	if err := h.uc.TransitionUserStatus(ctx, *request); err != nil {
		response.ErrorBuilder(err).Send(c)
		return
	}

	response.SuccessBuilder(nil).Send(c)
}
