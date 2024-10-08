package delivery

import (
	"net/http"
	"strconv"
	"time"

	"github.com/elyarsadig/studybud-go/configs"
	"github.com/elyarsadig/studybud-go/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (h *ApiHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "login.html", BaseTemplateData{})
}

func (h *ApiHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData{}
	ctx := r.Context()
	email := r.FormValue("email")
	password := r.FormValue("password")
	form := &domain.UserLoginForm{
		Email:    email,
		Password: password,
	}
	useCase := domain.Bridge[domain.UserUseCase](configs.USERS_DB_NAME, h.useCases)
	sessionKey, err := useCase.Login(ctx, form)
	if err != nil {
		h.handleError(w, err, "login.html", data)
		return
	}
	h.setCookie(w, sessionKey)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *ApiHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seesion, ok := h.extractSessionFromCookie(r)
	cookie := &http.Cookie{}
	if ok {
		err := h.redis.Remove(ctx, "session", seesion.SessionKey)
		if err != nil {
			h.logger.Error(err.Error())
			return
		}
	}
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *ApiHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "register.html", BaseTemplateData{})
}

func (h *ApiHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData{}
	ctx := r.Context()
	name := r.FormValue("name")
	userName := r.FormValue("username")
	email := r.FormValue("email")
	password1 := r.FormValue("password1")
	password2 := r.FormValue("password2")
	form := &domain.UserRegisterForm{
		Name:      name,
		Username:  userName,
		Email:     email,
		Password1: password1,
		Password2: password2,
	}
	useCase := domain.Bridge[domain.UserUseCase](configs.USERS_DB_NAME, h.useCases)
	sessionKey, err := useCase.RegisterUser(ctx, form)
	if err != nil {
		h.handleError(w, err, "register.html", data)
		return
	}
	h.setCookie(w, sessionKey)
	http.Redirect(w, r, "/user-update", http.StatusFound)
}

func (h *ApiHandler) Topics(w http.ResponseWriter, r *http.Request) {
	data := BaseTemplateData{}
	sessionValue, ok := h.extractSessionFromCookie(r)
	if ok {
		data = BaseTemplateData{
			AvatarURL:       sessionValue.Avatar,
			Username:        sessionValue.Username,
			IsAuthenticated: true,
		}
	}
	ctx := r.Context()
	useCase := domain.Bridge[domain.TopicUseCase](configs.TOPICS_DB_NAME, h.useCases)
	queryParams := r.URL.Query()
	name := queryParams.Get("q")
	var topics domain.Topics
	var err error
	if len(name) == 0 {
		topics, err = useCase.ListAllTopics(ctx)
		if err != nil {
			h.handleError(w, err, "topics.html", data)
			return
		}
	} else {
		topics, err = useCase.SearchTopicByName(ctx, name)
		if err != nil {
			h.handleError(w, err, "topics.html", data)
			return
		}
	}
	tmplData := Topics{
		BaseTemplateData: data,
		Topics:           topics,
	}
	h.renderTemplate(w, "topics.html", tmplData)
}

func (h *ApiHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	sessionValue, ok := h.extractSessionFromCookie(r)
	baseData := BaseTemplateData{
		Username:        sessionValue.Username,
		IsAuthenticated: ok,
		AvatarURL:       sessionValue.Avatar,
	}
	data := HomeTemplateData{
		BaseTemplateData: baseData,
	}
	queryParams := r.URL.Query()
	searchQuery := queryParams.Get("q")
	ctx := r.Context()
	topicUseCase := domain.Bridge[domain.TopicUseCase](configs.TOPICS_DB_NAME, h.useCases)
	roomUseCase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	messageUseCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	topics, err := topicUseCase.ListAllTopics(ctx)
	if err != nil {
		h.handleError(w, err, "home.html", baseData)
		return
	}
	data.TopicList = topics.List
	data.TopicsCount = topics.Count
	rooms, err := roomUseCase.ListRooms(ctx, searchQuery)
	if err != nil {
		h.handleError(w, err, "home.html", baseData)
		return
	}
	data.RoomCount = rooms.Count
	data.RoomList = rooms.List
	messages, err := messageUseCase.ListAllMessages(ctx)
	if err != nil {
		h.handleError(w, err, "home.html", baseData)
		return
	}
	data.MessageList = messages.MessageList
	h.renderTemplate(w, "home.html", data)
}

func (h *ApiHandler) CreateRoomPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionValue := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	baseData := BaseTemplateData{
		IsAuthenticated: true,
		Username:        sessionValue.Username,
		AvatarURL:       sessionValue.Avatar,
	}
	data := CreateRoomTemplateData{
		BaseTemplateData: baseData,
	}
	useCase := domain.Bridge[domain.TopicUseCase](configs.TOPICS_DB_NAME, h.useCases)
	topics, err := useCase.ListAllTopics(ctx)
	if err != nil {
		h.handleError(w, err, "room_form.html", baseData)
		return
	}
	data.TopicList = topics.List
	h.renderTemplate(w, "room_form.html", data)
}

func (h *ApiHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionValue := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	data := BaseTemplateData{
		AvatarURL:       sessionValue.Avatar,
		Username:        sessionValue.Username,
		IsAuthenticated: true,
	}
	roomForm := domain.RoomForm{
		TopicName:   r.FormValue("topic"),
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}
	useCase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	err := useCase.CreateRoom(ctx, roomForm)
	if err != nil {
		h.handleError(w, err, "room_form.html", data)
		return
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *ApiHandler) UpdateProfilePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionValue := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	useCase := domain.Bridge[domain.UserUseCase](configs.USERS_DB_NAME, h.useCases)
	user, err := useCase.GetUserByEmail(ctx, sessionValue.Email)
	if err != nil {
		h.handleError(w, err, "update_user.html", BaseTemplateData{})
		return
	}
	data := UpdateProfileTemplateData{
		BaseTemplateData: BaseTemplateData{
			IsAuthenticated: true,
			Username:        sessionValue.Username,
			AvatarURL:       sessionValue.Avatar,
		},
		Avatar:   user.Avatar,
		Username: user.Username,
		Name:     user.Name,
		Email:    user.Email,
		Bio:      user.Bio,
	}
	h.renderTemplate(w, "update_user.html", data)
}

func (h *ApiHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	updateUser, err := h.extractUserProfileUpdateForm(r)
	if err != nil {
		h.logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	useCase := domain.Bridge[domain.UserUseCase](configs.USERS_DB_NAME, h.useCases)
	sessionKey, err := useCase.UpdateInfo(ctx, &updateUser)
	if err != nil {
		h.handleError(w, err, "update_user.html", BaseTemplateData{})
		return
	}
	h.setCookie(w, sessionKey)
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *ApiHandler) DeleteMessagePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	request := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	baseData := BaseTemplateData{
		IsAuthenticated: true,
		AvatarURL:       request.Avatar,
		Username:        request.Username,
	}
	useCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	id := chi.URLParam(r, "id")
	message, err := useCase.GetUserMessage(ctx, id)
	if err != nil {
		h.handleError(w, err, "not_found.html", baseData)
		return
	}
	data := DeleteForm{
		BaseTemplateData: baseData,
		Obj:              message.Body,
	}
	h.renderTemplate(w, "delete.html", data)
}

func (h *ApiHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	useCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	err := useCase.Delete(ctx, id)
	if err != nil {
		h.handleError(w, err, "delete.html", BaseTemplateData{})
		return
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *ApiHandler) ActivitiesPage(w http.ResponseWriter, r *http.Request) {
	sessionValue, ok := h.extractSessionFromCookie(r)
	baseData := BaseTemplateData{
		AvatarURL:       sessionValue.Avatar,
		Username:        sessionValue.Username,
		IsAuthenticated: ok,
	}
	ctx := r.Context()
	useCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	messages, err := useCase.ListAllMessages(ctx)
	if err != nil {
		h.handleError(w, err, "activity.html", baseData)
		return
	}
	data := ActivitiesTemplateData{
		BaseTemplateData: baseData,
		MessageList:      messages.MessageList,
	}
	h.renderTemplate(w, "activity.html", data)
}

func (h *ApiHandler) UserProfilePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sv, ok := h.extractSessionFromCookie(r)
	baseData := BaseTemplateData{
		AvatarURL:       sv.Avatar,
		Username:        sv.Username,
		IsAuthenticated: ok,
	}
	data := UserProfileTemplateData{
		BaseTemplateData: baseData,
	}
	userID := chi.URLParam(r, "id")
	topicUC := domain.Bridge[domain.TopicUseCase](configs.TOPICS_DB_NAME, h.useCases)
	messageUC := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	userUC := domain.Bridge[domain.UserUseCase](configs.USERS_DB_NAME, h.useCases)
	roomUC := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	topics, err := topicUC.ListAllTopics(ctx)
	if err != nil {
		h.handleError(w, err, "profile.html", baseData)
		return
	}
	user, err := userUC.GetUserById(ctx, userID)
	if err != nil {
		h.handleError(w, err, "profile.html", baseData)
		return
	}
	rooms, err := roomUC.ListUserRooms(ctx, userID)
	if err != nil {
		h.handleError(w, err, "profile.html", baseData)
		return
	}
	messages, err := messageUC.ListUserMessages(ctx, userID)
	if err != nil {
		h.handleError(w, err, "profile.html", baseData)
		return
	}
	data.TopicList = topics.List
	data.TopicsCount = topics.Count
	data.User = user
	data.RoomList = rooms.List
	data.RoomCount = rooms.Count
	data.MessageList = messages.MessageList
	h.renderTemplate(w, "profile.html", data)
}

func (h *ApiHandler) RoomPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sv, ok := h.extractSessionFromCookie(r)
	baseData := BaseTemplateData{
		IsAuthenticated: ok,
		AvatarURL:       sv.Avatar,
		Username:        sv.Username,
	}
	roomID := chi.URLParam(r, "id")
	roomUseCase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	messageUseCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	room, err := roomUseCase.GetRoomById(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "room.html", baseData)
		return
	}
	participants, err := roomUseCase.ListRoomParticipants(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "room.html", baseData)
		return
	}
	messages, err := messageUseCase.ListRoomMessages(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "room.html", baseData)
		return
	}
	data := RoomTemplateData{
		BaseTemplateData: baseData,
		Room:             room,
		MessageList:      messages.MessageList,
		Participants:     participants,
	}
	h.renderTemplate(w, "room.html", data)
}

func (h *ApiHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	roomID, _ := strconv.Atoi(id)
	useCase := domain.Bridge[domain.MessageUseCase](configs.MESSAGES_DB_NAME, h.useCases)
	body := r.FormValue("body")
	message := &domain.Message{RoomID: uint(roomID), Body: body}
	err := useCase.CreateMessage(ctx, message)
	if err != nil {
		h.handleError(w, err, "room.html", BaseTemplateData{})
		return
	}
	http.Redirect(w, r, "/room/"+id, http.StatusFound)
}

func (h *ApiHandler) DeleteRoomPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	request := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	baseData := BaseTemplateData{
		IsAuthenticated: true,
		AvatarURL:       request.Avatar,
		Username:        request.Username,
	}
	usecase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	roomID := chi.URLParam(r, "id")
	room, err := usecase.GetUserRoom(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "not_found.html", baseData)
		return
	}
	data := DeleteForm{
		BaseTemplateData: baseData,
		Obj:              room.Name,
	}
	h.renderTemplate(w, "delete.html", data)
}

func (h *ApiHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roomID := chi.URLParam(r, "id")
	useCase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	err := useCase.DeleteUserRoom(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "room.html", BaseTemplateData{})
		return
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *ApiHandler) UpdateRoomPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sv := ctx.Value(configs.UserCtxKey).(domain.SessionValue)
	baseData := BaseTemplateData{
		IsAuthenticated: true,
		AvatarURL:       sv.Avatar,
		Username:        sv.Username,
	}
	roomID := chi.URLParam(r, "id")
	roomUsecase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	topicUsecase := domain.Bridge[domain.TopicUseCase](configs.TOPICS_DB_NAME, h.useCases)
	data := UpdateRoomTemplateData{
		BaseTemplateData: baseData,
	}
	room, err := roomUsecase.GetUserRoom(ctx, roomID)
	if err != nil {
		h.handleError(w, err, "room_form.html", baseData)
		return
	}
	topics, err := topicUsecase.ListAllTopics(ctx)
	if err != nil {
		h.handleError(w, err, "room_form.html", baseData)
		return
	}
	data.TopicList = topics.List
	data.Form = domain.RoomForm{
		TopicName:   room.Topic.Name,
		Name:        room.Name,
		Description: room.Description,
	}
	h.renderTemplate(w, "room_form.html", data)
}

func (h *ApiHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	roomID := chi.URLParam(r, "id")
	useCase := domain.Bridge[domain.RoomUseCase](configs.ROOMS_DB_NAME, h.useCases)
	roomForm := domain.RoomForm{
		Name:        r.FormValue("name"),
		TopicName:   r.FormValue("topic"),
		Description: r.FormValue("description"),
	}
	err := useCase.UpdateRoom(ctx, roomID, roomForm)
	if err != nil {
		h.handleError(w, err, "room_form.html", BaseTemplateData{})
		return
	}
	http.Redirect(w, r, "/home", http.StatusFound)
}
