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

package discord

import (
	"embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/apache/incubator-answer-plugins/util"
	"github.com/go-resty/resty/v2"

	discordI18n "github.com/HexmosTech/notification-discord/i18n"
	"github.com/apache/incubator-answer/plugin"
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
}

func (n *Notification) Info() plugin.Info {
	info := &util.Info{}
	info.GetInfo(Info)

	return plugin.Info{
		Name:        plugin.MakeTranslator(discordI18n.InfoName),
		SlugName:    info.SlugName,
		Description: plugin.MakeTranslator(discordI18n.InfoDescription),
		Author:      info.Author,
		Version:     info.Version,
		Link:        info.Link,
	}
}

// GetNewQuestionSubscribers returns the subscribers of the new question notification
func (n *Notification) GetNewQuestionSubscribers() (userIDs []string) {
	for userID, conf := range n.UserConfigCache.userConfigMapping {
		if conf.AllNewQuestions {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs
}

// Notify sends a notification to the user
func (n *Notification) Notify(msg plugin.NotificationMessage) {
	log.Debugf("try to send notification %+v", msg)

	if !n.Config.Notification {
		return
	}

	// get user config
	userConfig, err := n.getUserConfig(msg.ReceiverUserID)
	if err != nil {
		log.Errorf("get user config failed: %v", err)
		return
	}
	if userConfig == nil {
		log.Debugf("user %s has no config", msg.ReceiverUserID)
		return
	}

	// check if the notification is enabled
	switch msg.Type {
	case plugin.NotificationNewQuestion:
		if !userConfig.AllNewQuestions {
			log.Debugf("user %s not config the new question", msg.ReceiverUserID)
			return
		}
	case plugin.NotificationNewQuestionFollowedTag:
		if !userConfig.NewQuestionsForFollowingTags {
			log.Debugf("user %s not config the new question followed tag", msg.ReceiverUserID)
			return
		}
	default:
		if !userConfig.InboxNotifications {
			log.Debugf("user %s not config the inbox notification", msg.ReceiverUserID)
			return
		}
	}

	log.Debugf("user %s config the notification", msg.ReceiverUserID)

	if len(userConfig.WebhookURL) == 0 {
		log.Errorf("user %s has no webhook url", msg.ReceiverUserID)
		return
	}

	notificationMsg := renderNotification(msg)
	log.Infof("notification message: %s", notificationMsg)
	// no need to send empty message
	if len(notificationMsg) == 0 {
		log.Debugf("this type of notification will be drop, the type is %s", msg.Type)
		return
	}

	// Create a Resty Client
	client := resty.New()
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(NewWebhookReq(notificationMsg)).
		Post(userConfig.WebhookURL)

	if err != nil {
		log.Errorf("send message failed: %v %v", err, resp)
	} else {
		log.Infof("send message to %s success, resp: %s", msg.ReceiverUserID, resp.String())
	}
}

func renderNotification(msg plugin.NotificationMessage) string {
	// lang := i18n.Language(msg.ReceiverLang)
	switch msg.Type {
	case plugin.NotificationUpdateQuestion:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "updated answer", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationAnswerTheQuestion:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "answered your question", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationAcceptAnswer:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "accepted your answer", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationCommentQuestion:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "commented on your question", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationCommentAnswer:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "commented on your answer", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationReplyToYou:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "replied to your comment", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationMentionYou:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "mentioned you in a comment", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationInvitedYouToAnswer:
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "invited you to answer", msg.QuestionTitle, msg.QuestionUrl)
	case plugin.NotificationNewQuestion, plugin.NotificationNewQuestionFollowedTag:
		msg.QuestionTags = strings.Join(strings.Split(msg.QuestionTags, ","), ", ")
		return formatMarkdownNotification(msg.TriggerUserDisplayName, "asked a new question", msg.QuestionTitle, msg.QuestionUrl)
	}
	return ""
}

// Add this new helper function
func formatMarkdownNotification(username, action, title, url string) string {
	return fmt.Sprintf("[%s](%s) %s [%s](%s)", username, getUserURL(username), action, title, url)
}

// Add this helper function to construct the user URL
func getUserURL(username string) string {
	return fmt.Sprintf("https://answer.apps.hexmos.com/users/%s", url.PathEscape(username))
}
