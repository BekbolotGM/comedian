package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/language"
)

//Translate struct makes translation data
type Translate struct {
	UserExist            string
	AddUserNoStandupTime string
	AddUser              string
	AddAdmin             string
	AccessDenied         string

	DeleteUser       string
	DeleteAdmin      string
	ListNoStandupers string
	ListNoAdmins     string
	ListStandupers   string
	ListAdmins       string

	AddStandupTimeNoUsers      string
	AddStandupTime             string
	RemoveStandupTimeWithUsers string
	RemoveStandupTime          string
	ShowNoStandupTime          string
	ShowStandupTime            string
	WrongNArgs                 string

	NoWorklogs          string
	NoCommits           string
	NoStandup           string
	HasWorklogs         string
	HasCommits          string
	HasStandup          string
	IsRook              string
	IsRookMonday        string
	NotifyAllDone       string
	NotifyNotAll        string
	NotifyManagerNotAll string
	NotifyUsersWarning  string
	NotifyDirectMessage string

	ReportByProjectAndUser       string
	ReportOnProjectHead          string
	ReportOnProjectCollectorData string
	ReportOnUserHead             string
	ReportOnProjectAndUserHead   string
	ReportNoData                 string
	ReportDate                   string
	ReportStandupFromUser        string
	ReportIgnoredStandup         string
	ReportShowChannel            string
	ReportCollectorDataUser      string
	UserDidNotStandup            string
	UserDidStandup               string
	UserDidNotStandupInChannel   string
	UserDidStandupInChannel      string
	PMAssigned                   string
	PMRemoved                    string

	DateError1 string
	DateError2 string

	HelloManager    string
	StandupAccepted string

	P1 string
	P2 string
	P3 string

	Y1 string
	Y2 string
	Y3 string
	Y4 string

	T1 string
	T2 string
	T3 string

	WrongUsernameError string
}

// GetTranslation sets translation files for config
func GetTranslation(lang string) (Translate, error) {
	bundle := &i18n.Bundle{DefaultLanguage: language.English}
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	wd, err := os.Getwd()
	if err != nil {
		return Translate{}, err
	}
	if !strings.HasSuffix(wd, "comedian") {
		wd = filepath.Dir(wd)
	}
	_, err = bundle.LoadMessageFile(fmt.Sprintf("%s/config/ru.toml", wd))
	if err != nil {
		return Translate{}, err
	}
	_, err = bundle.LoadMessageFile(fmt.Sprintf("%s/config/en.toml", wd))
	if err != nil {
		return Translate{}, err
	}
	localizer := i18n.NewLocalizer(bundle, lang)
	if err != nil {
		logrus.Errorf("slack: GetLocalizer failed: %v\n", err)
		return Translate{}, err
	}
	m := make(map[string]string)
	r := []string{
		"noWorklogs", "noCommits", "noStandup", "hasWorklogs",
		"hasCommits", "hasStandup", "isRook", "notifyAllDone",
		"notifyNotAll", "notifyManagerNotAll", "notifyUsersWarning",
		"notifyDirectMessage",
		"reportByProjectAndUser", "reportOnProjectHead", "reportOnProjectCollectorData", "reportOnUserHead",
		"reportOnProjectAndUserHead", "reportNoData", "reportDate",
		"reportStandupFromUser", "reportIgnoredStandup", "reportShowChannel",
		"reportCollectorDataUser",
		"helloManager", "standupAccepted",
		"p1", "p2", "p3",
		"y1", "y2", "y3", "y4",
		"t1", "t2", "t3",
		"userExist",
		"addUserNoStandupTime",
		"addUser",
		"addAdmin",
		"accessDenied",
		"deleteUser",
		"deleteAdmin",
		"listNoStandupers",
		"listNoAdmins",
		"listStandupers",
		"listAdmins",
		"addStandupTimeNoUsers",
		"addStandupTime",
		"removeStandupTimeWithUsers",
		"removeStandupTime",
		"showNoStandupTime",
		"showStandupTime",
		"wrongNArgs",
		"dateError1", "dateError2",
		"userDidNotStandup", "userDidStandup",
		"userDidNotStandupInChannel", "userDidStandupInChannel",
		"PMAssigned",
		"PMRemoved",
		"isRookMonday",
		"wrongUsernameError",
	}

	for _, t := range r {
		translated, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: t})
		if err != nil {
			logrus.Errorf("slack: Localize failed: %v\n", err)
			return Translate{}, err
		}
		m[t] = translated
	}

	t := Translate{
		UserExist:            m["userExist"],
		AddUserNoStandupTime: m["addUserNoStandupTime"],
		AddUser:              m["addUser"],
		AddAdmin:             m["addAdmin"],
		AccessDenied:         m["accessDenied"],

		DeleteUser:       m["deleteUser"],
		DeleteAdmin:      m["deleteAdmin"],
		ListNoStandupers: m["listNoStandupers"],
		ListNoAdmins:     m["listNoAdmins"],
		ListStandupers:   m["listStandupers"],
		ListAdmins:       m["listAdmins"],

		AddStandupTimeNoUsers:        m["addStandupTimeNoUsers"],
		AddStandupTime:               m["addStandupTime"],
		RemoveStandupTimeWithUsers:   m["removeStandupTimeWithUsers"],
		RemoveStandupTime:            m["removeStandupTime"],
		ShowNoStandupTime:            m["showNoStandupTime"],
		ShowStandupTime:              m["showStandupTime"],
		WrongNArgs:                   m["wrongNArgs"],
		NoWorklogs:                   m["noWorklogs"],
		NoCommits:                    m["noCommits"],
		NoStandup:                    m["noStandup"],
		HasWorklogs:                  m["hasWorklogs"],
		HasCommits:                   m["hasCommits"],
		HasStandup:                   m["hasStandup"],
		IsRook:                       m["isRook"],
		IsRookMonday:                 m["isRookMonday"],
		NotifyAllDone:                m["notifyAllDone"],
		NotifyNotAll:                 m["notifyNotAll"],
		NotifyManagerNotAll:          m["notifyManagerNotAll"],
		NotifyUsersWarning:           m["notifyUsersWarning"],
		NotifyDirectMessage:          m["notifyDirectMessage"],
		ReportByProjectAndUser:       m["reportByProjectAndUser"],
		ReportOnProjectHead:          m["reportOnProjectHead"],
		ReportOnProjectCollectorData: m["reportOnProjectCollectorData"],
		ReportOnUserHead:             m["reportOnUserHead"],
		ReportOnProjectAndUserHead:   m["reportOnProjectAndUserHead"],
		ReportNoData:                 m["reportNoData"],
		ReportDate:                   m["reportDate"],
		ReportStandupFromUser:        m["reportStandupFromUser"],
		ReportIgnoredStandup:         m["reportIgnoredStandup"],
		ReportShowChannel:            m["reportShowChannel"],
		ReportCollectorDataUser:      m["reportCollectorDataUser"],
		DateError1:                   m["dateError1"],
		DateError2:                   m["dateError2"],
		HelloManager:                 m["helloManager"],
		StandupAccepted:              m["standupAccepted"],
		UserDidNotStandup:            m["userDidNotStandup"],
		UserDidStandup:               m["userDidStandup"],
		UserDidNotStandupInChannel:   m["userDidNotStandupInChannel"],
		UserDidStandupInChannel:      m["userDidStandupInChannel"],
		PMAssigned:                   m["PMAssigned"],
		PMRemoved:                    m["PMRemoved"],

		P1: m["p1"],
		P2: m["p2"],
		P3: m["p3"],

		Y1: m["y1"],
		Y2: m["y2"],
		Y3: m["y3"],
		Y4: m["y4"],

		T1: m["t1"],
		T2: m["t2"],
		T3: m["t3"],

		WrongUsernameError: m["wrongUsernameError"],
	}

	return t, nil
}
