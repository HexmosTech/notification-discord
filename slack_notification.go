/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package slack

import (
	"embed"
	"github.com/apache/incubator-answer-plugins/util"
	"github.com/go-resty/resty/v2"
	"strings"

	slackI18n "github.com/apache/incubator-answer-plugins/notification-slack/i18n"
	"github.com/apache/incubator-answer/plugin"
	"github.com/segmentfault/pacman/i18n"
	"github.com/segmentfault/pacman/log"
)

//go:embed  info.yaml
var Info embed.FS

type Notification struct {
	Config          *NotificationConfig
	UserConfigCache *UserConfigCache
}

func init() {
	uc := &Notification{
		Config:          &NotificationConfig{},
		UserConfigCache: NewUserConfigCache(),
	}
	plugin.Register(uc)
	log.Debugf("Slack notification plugin initialized")
}

func (n *Notification) Info() plugin.Info {
	info := &util.Info{}
	info.GetInfo(Info)

	log.Debugf("Retrieving Slack notification plugin info")
	return plugin.Info{
		Name:        plugin.MakeTranslator(slackI18n.InfoName),
		SlugName:    info.SlugName,
		Description: plugin.MakeTranslator(slackI18n.InfoDescription),
		Author:      info.Author,
		Version:     info.Version,
		Link:        info.Link,
	}
}

// GetNewQuestionSubscribers returns the subscribers of the new question notification
func (n *Notification) GetNewQuestionSubscribers() (userIDs []string) {
	log.Debugf("Getting new question subscribers")
	for userID, conf := range n.UserConfigCache.userConfigMapping {
		if conf.AllNewQuestions {
			userIDs = append(userIDs, userID)
		}
	}
	log.Debugf("Found %d subscribers for new questions", len(userIDs))
	return userIDs
}

// Notify sends a notification to the user
func (n *Notification) Notify(msg plugin.NotificationMessage) {
	log.Debugf("Attempting to send notification: %+v", msg)

	if !n.Config.Notification {
		log.Debugf("Notifications are disabled in config")
		return
	}

	// get user config
	userConfig, err := n.getUserConfig(msg.ReceiverUserID)
	if err != nil {
		log.Errorf("Failed to get user config: %v", err)
		return
	}
	if userConfig == nil {
		log.Debugf("User %s has no config", msg.ReceiverUserID)
		return
	}

	// check if the notification is enabled
	switch msg.Type {
	case plugin.NotificationNewQuestion:
		if !userConfig.AllNewQuestions {
			log.Debugf("User %s has not configured new question notifications", msg.ReceiverUserID)
			return
		}
	case plugin.NotificationNewQuestionFollowedTag:
		if !userConfig.NewQuestionsForFollowingTags {
			log.Debugf("User %s has not configured new question followed tag notifications", msg.ReceiverUserID)
			return
		}
	default:
		if !userConfig.InboxNotifications {
			log.Debugf("User %s has not configured inbox notifications", msg.ReceiverUserID)
			return
		}
	}

	log.Debugf("User %s has configured the notification", msg.ReceiverUserID)

	if len(userConfig.WebhookURL) == 0 {
		log.Errorf("User %s has no webhook URL", msg.ReceiverUserID)
		return
	}

	notificationMsg := renderNotification(msg)
	// no need to send empty message
	if len(notificationMsg) == 0 {
		log.Debugf("Empty notification message for type %s, dropping", msg.Type)
		return
	}

	log.Debugf("Sending message to %s: %s", msg.ReceiverUserID, notificationMsg)

	// Create a Resty Client
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(NewWebhookReq(notificationMsg)).
		Post(userConfig.WebhookURL)

	if err != nil {
		log.Errorf("Failed to send message: %v, Response: %v", err, resp)
	} else {
		log.Infof("Successfully sent message to %s, Response: %s", msg.ReceiverUserID, resp.String())
	}
}

func renderNotification(msg plugin.NotificationMessage) string {
	log.Debugf("Rendering notification for message type: %s", msg.Type)
	lang := i18n.Language(msg.ReceiverLang)
	var result string
	switch msg.Type {
	case plugin.NotificationUpdateQuestion:
		result = plugin.TranslateWithData(lang, slackI18n.TplUpdateQuestion, msg)
	case plugin.NotificationAnswerTheQuestion:
		result = plugin.TranslateWithData(lang, slackI18n.TplAnswerTheQuestion, msg)
	case plugin.NotificationUpdateAnswer:
		result = plugin.TranslateWithData(lang, slackI18n.TplUpdateAnswer, msg)
	case plugin.NotificationAcceptAnswer:
		result = plugin.TranslateWithData(lang, slackI18n.TplAcceptAnswer, msg)
	case plugin.NotificationCommentQuestion:
		result = plugin.TranslateWithData(lang, slackI18n.TplCommentQuestion, msg)
	case plugin.NotificationCommentAnswer:
		result = plugin.TranslateWithData(lang, slackI18n.TplCommentAnswer, msg)
	case plugin.NotificationReplyToYou:
		result = plugin.TranslateWithData(lang, slackI18n.TplReplyToYou, msg)
	case plugin.NotificationMentionYou:
		result = plugin.TranslateWithData(lang, slackI18n.TplMentionYou, msg)
	case plugin.NotificationInvitedYouToAnswer:
		result = plugin.TranslateWithData(lang, slackI18n.TplInvitedYouToAnswer, msg)
	case plugin.NotificationNewQuestion, plugin.NotificationNewQuestionFollowedTag:
		msg.QuestionTags = strings.Join(strings.Split(msg.QuestionTags, ","), ", ")
		result = plugin.TranslateWithData(lang, slackI18n.TplNewQuestion, msg)
	default:
		log.Debugf("Unknown notification type: %s", msg.Type)
	}
	log.Debugf("Rendered notification: %s", result)
	return result
}
