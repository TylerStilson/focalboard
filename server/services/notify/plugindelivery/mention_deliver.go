// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package plugindelivery

import (
	"fmt"

	"github.com/mattermost/focalboard/server/services/notify"
	"github.com/mattermost/focalboard/server/utils"

	mm_model "github.com/mattermost/mattermost-server/v6/model"
)

// MentionDeliver notifies a user they have been mentioned in a blockv ia the plugin API.
func (pd *PluginDelivery) MentionDeliver(mentionedUser *mm_model.User, extract string, evt notify.BlockChangeEvent) (string, error) {
	author, err := pd.api.GetUserByID(evt.ModifiedBy.UserID)
	if err != nil {
		return "", fmt.Errorf("cannot find user: %w", err)
	}

	channel, err := pd.api.GetDirectChannel(mentionedUser.Id, pd.botID)
	if err != nil {
		return "", fmt.Errorf("cannot get direct channel: %w", err)
	}
	link := utils.MakeCardLink(pd.serverRoot, evt.Board.TeamID, evt.Board.ID, evt.Card.ID)

	post := &mm_model.Post{
		UserId:    pd.botID,
		ChannelId: channel.Id,
		Message:   formatMessage(author.Username, extract, evt.Card.Title, link, evt.BlockChanged),
	}
	return mentionedUser.Id, pd.api.CreatePost(post)
}
