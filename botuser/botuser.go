package botuser

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"gitlab.com/team-monitoring/comedian/model"
	"gitlab.com/team-monitoring/comedian/storage"
	"gitlab.com/team-monitoring/comedian/translation"
)

var (
	typeMessage       = ""
	typeEditMessage   = "message_changed"
	typeDeleteMessage = "message_deleted"
	//Dry is used to implement Dry run for bot methods
	Dry bool
)

const (
	superAdminAccess  = 1
	adminAccess       = 2
	pmAccess          = 3
	regularUserAccess = 4
	noAccess          = 0
)

// Bot struct used for storing and communicating with slack api
type Bot struct {
	slack      *slack.Client
	properties model.BotSettings
	db         storage.Storage
	bundle     *i18n.Bundle
	wg         sync.WaitGroup
	QuitChan   chan struct{}
}

//New creates new Bot instance
func New(bundle *i18n.Bundle, settings model.BotSettings, db storage.Storage) *Bot {
	quit := make(chan struct{})

	bot := &Bot{}
	bot.slack = slack.New(settings.AccessToken)
	bot.properties = settings
	bot.db = db
	bot.bundle = bundle
	bot.QuitChan = quit

	return bot
}

//Start updates Users list and launches notifications
func (bot *Bot) Start() {
	log.Info("Bot started: ", bot.properties)

	err := bot.UpdateUsersList()
	if err != nil {
		log.Errorf("UpdateUsersList failed: %v", err)
	}

	bot.wg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Second * 60).C
		for {
			select {
			case <-ticker:
				bot.NotifyChannels(time.Now())
				bot.SetStandupsCounterToZero()
				if Dry {
					break
				}
			case <-bot.QuitChan:
				log.Infof("Mission completed, gbye!!! Truly yours, %v bot", bot.properties.TeamName)
				bot.wg.Done()
				return
			}
		}
	}()
}

//Stop closes bot QuitChan making bot goroutine to exit
func (bot *Bot) Stop() {
	close(bot.QuitChan)
}

func (bot *Bot) SetStandupsCounterToZero() error {
	if (time.Now().Hour() != 23) && (time.Now().Minute() != 59) {
		return nil
	}
	log.Info("Started to set submitted standups to 0 for all standupers")
	standupers, err := bot.db.ListStandupersByTeamID(bot.properties.TeamID)
	if err != nil {
		return err
	}
	for _, standuper := range standupers {
		standuper.SubmittedStandupToday = false
		bot.db.UpdateStanduper(standuper)
	}
	return nil
}

//HandleCallBackEvent handles different callback events from Slack Event Subscription list
func (bot *Bot) HandleCallBackEvent(event *json.RawMessage) error {
	ev := map[string]string{}
	data, err := event.MarshalJSON()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}

	switch ev["type"] {
	case "app_mention":
		message := &slack.MessageEvent{}
		if err := json.Unmarshal(data, message); err != nil {
			return err
		}
		return bot.HandleMessage(message)

	case "member_joined_channel":
		join := &slack.MemberJoinedChannelEvent{}
		if err := json.Unmarshal(data, join); err != nil {
			return err
		}
		//need to check if join.Team is teamID, not a teamName
		_, err = bot.HandleJoin(join.Channel, join.Team)
		return err
	case "app_uninstalled":
		bot.Stop()
		err := bot.db.DeleteBotSettings(bot.properties.TeamID)
		if err != nil {
			return err
		}
	default:
		log.WithFields(log.Fields{"event": event}).Warning("unrecognized event!")
		return nil
	}

	return nil
}

//HandleMessage handles slack message event
func (bot *Bot) HandleMessage(msg *slack.MessageEvent) error {
	msg.Team = bot.properties.TeamID
	switch msg.SubType {
	case typeMessage:
		return bot.handleNewMessage(msg)
	case typeEditMessage:
		return bot.handleEditMessage(msg)
	case typeDeleteMessage:
		return bot.handleDeleteMessage(msg)
	}
	return nil
}

func (bot *Bot) handleNewMessage(msg *slack.MessageEvent) error {
	if !strings.Contains(msg.Msg.Text, bot.properties.UserID) {
		return nil
	}

	problem := bot.analizeStandup(msg.Msg.Text)
	if problem != "" {
		return bot.SendEphemeralMessage(msg.Channel, msg.User, problem)
	}

	standuper, err := bot.db.FindStansuperByUserID(msg.User, msg.Channel)
	if err != nil {
		log.WithFields(log.Fields{"channel": msg.Channel, "error": err, "user": msg.User}).Warning("Non standuper submitted standup")
	}

	if standuper.SubmittedStandupToday || bot.submittedStandupToday(msg.User) {
		payload := translation.Payload{bot.bundle, bot.properties.Language, "OneStandupPerDay", 0, nil}
		oneStandupPerDay, err := translation.Translate(payload)
		if err != nil {
			log.WithFields(log.Fields{
				"TeamName":     bot.properties.TeamName,
				"Language":     payload.Lang,
				"MessageID":    payload.MessageID,
				"Count":        payload.Count,
				"TemplateData": payload.TemplateData,
			}).Error("Failed to translate message!")
		}
		bot.SendEphemeralMessage(msg.Channel, msg.User, oneStandupPerDay)
		return nil
	}
	standup, err := bot.db.CreateStandup(model.Standup{
		TeamID:    msg.Team,
		ChannelID: msg.Channel,
		UserID:    msg.User,
		Comment:   msg.Msg.Text,
		MessageTS: msg.Msg.Timestamp,
	})
	if err != nil {
		return err
	}
	log.Infof("Standup created #id:%v\n", standup.ID)
	item := slack.ItemRef{
		Channel:   msg.Channel,
		Timestamp: msg.Msg.Timestamp,
		File:      "",
		Comment:   "",
	}
	err = bot.slack.AddReaction("heavy_check_mark", item)
	if err != nil {
		log.WithFields(log.Fields{
			"TeamName": bot.properties.TeamName,
			"Item":     item,
		}).Error("Failed to AddReaction!")
	}

	standuper.SubmittedStandupToday = true
	_, err = bot.db.UpdateStanduper(standuper)
	if err != nil {
		return err
	}
	return nil
}

func (bot *Bot) handleEditMessage(msg *slack.MessageEvent) error {
	if !strings.Contains(msg.SubMessage.Text, bot.properties.UserID) {
		return nil
	}

	problem := bot.analizeStandup(msg.SubMessage.Text)
	if problem != "" {
		return bot.SendEphemeralMessage(msg.Channel, msg.SubMessage.User, problem)
	}

	standup, err := bot.db.SelectStandupByMessageTS(msg.SubMessage.Timestamp)
	if err == nil {
		standup.Comment = msg.SubMessage.Text
		st, err := bot.db.UpdateStandup(standup)
		if err != nil {
			return err
		}
		log.Infof("Standup updated #id:%v\n", st.ID)
		return nil
	}
	standuper, err := bot.db.FindStansuperByUserID(msg.User, msg.Channel)
	if err != nil {
		log.WithFields(log.Fields{"channel": msg.Channel, "error": err, "user": msg.User}).Warning("Non standuper submitted standup")
	}

	if standuper.SubmittedStandupToday || bot.submittedStandupToday(msg.SubMessage.User) {
		payload := translation.Payload{bot.bundle, bot.properties.Language, "OneStandupPerDay", 0, nil}
		oneStandupPerDay, err := translation.Translate(payload)
		if err != nil {
			log.WithFields(log.Fields{
				"TeamName":     bot.properties.TeamName,
				"Language":     payload.Lang,
				"MessageID":    payload.MessageID,
				"Count":        payload.Count,
				"TemplateData": payload.TemplateData,
			}).Error("Failed to translate message!")
		}
		bot.SendEphemeralMessage(msg.Channel, msg.SubMessage.User, oneStandupPerDay)
		return nil
	}

	standup, err = bot.db.CreateStandup(model.Standup{
		TeamID:    msg.Team,
		ChannelID: msg.Channel,
		UserID:    msg.SubMessage.User,
		Comment:   msg.SubMessage.Text,
		MessageTS: msg.SubMessage.Timestamp,
	})
	if err != nil {
		return err
	}

	log.Infof("Standup created #id:%v\n", standup.ID)

	item := slack.ItemRef{
		Channel:   msg.Channel,
		Timestamp: msg.SubMessage.Timestamp,
		File:      "",
		Comment:   "",
	}
	err = bot.slack.AddReaction("heavy_check_mark", item)
	if err != nil {
		log.WithFields(log.Fields{
			"TeamName": bot.properties.TeamName,
			"Item":     item,
		}).Error("Failed to AddReaction!")
	}
	standuper.SubmittedStandupToday = true
	_, err = bot.db.UpdateStanduper(standuper)
	if err != nil {
		return err
	}
	return nil
}

func (bot *Bot) handleDeleteMessage(msg *slack.MessageEvent) error {
	standup, err := bot.db.SelectStandupByMessageTS(msg.DeletedTimestamp)
	if err != nil {
		return err
	}
	return bot.db.DeleteStandup(standup.ID)
}

func (bot *Bot) submittedStandupToday(userID string) bool {
	log.Info("Start checking if user submitted standup today!", userID)
	standup, err := bot.db.SelectLatestStandupByUser(userID)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("standup.Modified.Day() == time.Now().Day() ", standup.Modified.Day() == time.Now().Day())
	if standup.Modified.Day() == time.Now().Day() {
		return true
	}
	return false
}

func (bot *Bot) analizeStandup(message string) string {
	message = strings.ToLower(message)

	mentionsYesterdayWork := false
	yesterdayWorkKeys := []string{"yesterday", "friday", "monday", "tuesday", "wednesday", "thursday", "saturday", "sunday", "completed", "вчера", "пятниц", "делал", "сделано", "понедельник", "вторник", "сред", "четверг", "суббот", "воскресенье"}
	for _, work := range yesterdayWorkKeys {
		if strings.Contains(message, work) {
			mentionsYesterdayWork = true
		}
	}

	if !mentionsYesterdayWork {
		payload := translation.Payload{bot.bundle, bot.properties.Language, "StandupHandleNoYesterdayWorkMentioned", 0, nil}
		problem, err := translation.Translate(payload)
		if err != nil {
			log.WithFields(log.Fields{
				"TeamName":     bot.properties.TeamName,
				"Language":     payload.Lang,
				"MessageID":    payload.MessageID,
				"Count":        payload.Count,
				"TemplateData": payload.TemplateData,
			}).Error("Failed to translate message!")

		}
		return problem
	}

	mentionsTodayPlans := false
	todayPlansKeys := []string{"today", "going", "plan", "сегодня", "собираюсь", "план"}
	for _, plan := range todayPlansKeys {
		if strings.Contains(message, plan) {
			mentionsTodayPlans = true
		}
	}
	if !mentionsTodayPlans {
		payload := translation.Payload{bot.bundle, bot.properties.Language, "StandupHandleNoTodayPlansMentioned", 0, nil}
		problem, err := translation.Translate(payload)
		if err != nil {
			log.WithFields(log.Fields{
				"TeamName":     bot.properties.TeamName,
				"Language":     payload.Lang,
				"MessageID":    payload.MessageID,
				"Count":        payload.Count,
				"TemplateData": payload.TemplateData,
			}).Error("Failed to translate message!")

		}
		return problem
	}

	mentionsProblem := false

	problemKeys := []string{"problem", "difficult", "stuck", "question", "issue", "block", "проблем", "трудност", "затруднени", "вопрос", "блок"}
	for _, problem := range problemKeys {
		if strings.Contains(message, problem) {
			mentionsProblem = true
		}
	}
	if !mentionsProblem {
		payload := translation.Payload{bot.bundle, bot.properties.Language, "StandupHandleNoProblemsMentioned", 0, nil}
		problem, err := translation.Translate(payload)
		if err != nil {
			log.WithFields(log.Fields{
				"TeamName":     bot.properties.TeamName,
				"Language":     payload.Lang,
				"MessageID":    payload.MessageID,
				"Count":        payload.Count,
				"TemplateData": payload.TemplateData,
			}).Error("Failed to translate message!")

		}
		return problem
	}

	return ""
}

// SendMessage posts a message in a specified channel visible for everyone
func (bot *Bot) SendMessage(channel, message string, attachments []slack.Attachment) error {
	if Dry {
		return nil
	}
	_, _, err := bot.slack.PostMessage(channel, message, slack.PostMessageParameters{
		Attachments: attachments,
	})
	return err
}

// SendEphemeralMessage posts a message in a specified channel which is visible only for selected user
func (bot *Bot) SendEphemeralMessage(channel, user, message string) error {
	if Dry {
		return nil
	}
	_, err := bot.slack.PostEphemeral(
		channel,
		user,
		slack.MsgOptionText(message, true),
	)
	return err
}

// SendUserMessage Direct Message specific user
func (bot *Bot) SendUserMessage(userID, message string) error {
	if Dry {
		return nil
	}
	_, _, channelID, err := bot.slack.OpenIMChannel(userID)
	if err != nil {
		return err
	}
	return bot.SendMessage(channelID, message, nil)
}

//HandleJoin handles comedian joining channel
func (bot *Bot) HandleJoin(channelID, teamID string) (model.Channel, error) {
	newChannel := model.Channel{}
	newChannel, err := bot.db.SelectChannel(channelID)
	if err == nil {
		return newChannel, nil
	}

	channel, err := bot.slack.GetConversationInfo(channelID, true)
	if err != nil {
		return newChannel, err
	}
	newChannel, err = bot.db.CreateChannel(model.Channel{
		TeamID:      teamID,
		ChannelName: channel.Name,
		ChannelID:   channel.ID,
		StandupTime: int64(0),
	})
	if err != nil {
		return newChannel, err
	}
	return newChannel, nil
}

//ImplementCommands implements slash commands such as adding users and managing deadlines
func (bot *Bot) ImplementCommands(channelID, command, params string, accessLevel int) string {

	switch command {
	case "add":
		return bot.addCommand(accessLevel, channelID, params)
	case "show":
		return bot.showCommand(channelID, params)
	case "remove":
		return bot.deleteCommand(accessLevel, channelID, params)
	case "add_deadline":
		return bot.addTime(accessLevel, channelID, params)
	case "remove_deadline":
		return bot.removeTime(accessLevel, channelID)
	case "show_deadline":
		return bot.showTime(channelID)
	default:
		return bot.DisplayHelpText("")
	}
}

//GetAccessLevel returns access level to figure out if a user can use slash command
func (bot *Bot) GetAccessLevel(userID, channelID string) (int, error) {
	user, err := bot.db.SelectUser(userID)
	if err != nil {
		return noAccess, err
	}
	if user.IsAdmin() {
		return adminAccess, nil
	}

	standuper, err := bot.db.FindStansuperByUserID(userID, channelID)
	if err != nil {
		return regularUserAccess, nil
	}

	if standuper.IsPM() {
		return pmAccess, nil
	}

	return regularUserAccess, nil
}

//UpdateUsersList updates users in workspace
func (bot *Bot) UpdateUsersList() error {
	users, err := bot.slack.GetUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		err := bot.updateUser(user)
		if err != nil {
			log.WithFields(log.Fields{"user": user, "bot": bot, "error": err}).Error("updateUser failed")
		}
	}
	return nil
}

//SendMessageToSuperAdmins messages all users in workspace
func (bot *Bot) SendMessageToSuperAdmins(message string) error {
	users, err := bot.db.ListUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		if user.TeamID == bot.properties.TeamID && user.Role == "super-admin" {
			bot.SendUserMessage(user.UserID, message)
		}
	}
	return nil
}

func (bot *Bot) updateUser(user slack.User) error {
	if user.IsBot || user.Name == "slackbot" {
		return nil
	}

	u, err := bot.db.SelectUser(user.ID)
	if err != nil && !user.Deleted {
		if user.IsOwner || user.IsPrimaryOwner {
			u, err = bot.db.CreateUser(model.User{
				TeamID:   user.TeamID,
				UserName: user.Name,
				UserID:   user.ID,
				Role:     "super-admin",
				RealName: user.RealName,
			})
			if err != nil {
				return err
			}
			return nil
		}
		if user.IsAdmin {
			u, err = bot.db.CreateUser(model.User{
				TeamID:   user.TeamID,
				UserName: user.Name,
				UserID:   user.ID,
				Role:     "admin",
				RealName: user.RealName,
			})
			if err != nil {
				return err
			}
			return nil
		}

		u, err = bot.db.CreateUser(model.User{
			TeamID:   user.TeamID,
			UserName: user.Name,
			UserID:   user.ID,
			Role:     "",
			RealName: user.RealName,
		})
		if err != nil {
			return err
		}
	}

	if !user.Deleted {
		u.UserName = user.Name
		if user.IsAdmin {
			u.Role = "admin"
		}
		if user.IsOwner || user.IsPrimaryOwner {
			u.Role = "super-admin"
		}
		u.RealName = user.RealName
		u.TeamID = user.TeamID
		_, err = bot.db.UpdateUser(u)
		if err != nil {
			return err
		}
	}

	if user.Deleted {
		err := bot.db.DeleteUser(u.ID)
		if err != nil {
			return err
		}

		standupers, err := bot.db.ListStandupers()
		if err != nil {
			return err
		}
		for _, standuper := range standupers {
			if u.UserID == standuper.UserID {
				err := bot.db.DeleteStanduper(standuper.ID)
				if err != nil {
					log.WithFields(log.Fields{"user": user, "bot": bot, "standuper": standuper, "error": err}).Error("DeleteStanduper failed")
				}
			}
		}
	}

	return nil
}

//Suits returns true if found desired bot properties
func (bot *Bot) Suits(team string) bool {
	return team == bot.properties.TeamID || team == bot.properties.TeamName
}

//Settings just returns bot settings
func (bot *Bot) Settings() model.BotSettings {
	return bot.properties
}

//SetProperties updates bot settings
func (bot *Bot) SetProperties(settings model.BotSettings) model.BotSettings {
	bot.properties = settings
	return bot.properties
}
