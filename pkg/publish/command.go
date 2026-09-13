// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package publish

import (
	"context"
	"net/url"

	"github.com/bborbe/errors"
	core "github.com/bborbe/notification"
	"github.com/bborbe/notification/command/notification"
)

// BuildCommand builds the notification-publish command for one matched task event.
//
// The command is typed agent-escalation and carries no target: the deployed
// routing table owns the channel decision, and a non-nil target would override
// it. Nothing is sent and nothing is validated here; the caller validates
// before sending.
func BuildCommand(
	ctx context.Context,
	event TaskEvent,
) (notification.NotificationPublishCommand, error) {
	deeplink, err := buildDeeplink(ctx, event)
	if err != nil {
		return notification.NotificationPublishCommand{}, errors.Wrapf(ctx, err, "build deeplink")
	}
	return notification.NotificationPublishCommand{
		Type:   core.AgentEscalationNotificationType,
		Target: nil,
		Message: core.NotificationMessagef(
			"Task %s is %s (phase %s, assignee %s)\n%s",
			event.TaskName,
			event.Status,
			event.Phase,
			event.Assignee,
			deeplink,
		),
		Metadata: map[string]string{
			"taskName": event.TaskName,
			"phase":    event.Phase,
			"assignee": event.Assignee,
			"vault":    event.Vault,
		},
	}, nil
}

// buildDeeplink returns the Obsidian deeplink to the task note inside its vault.
// The vault and the file value are percent-encoded so a space becomes %20 and
// never +; the slash between the tasks dir and the file name is preserved
// because each part is encoded on its own.
func buildDeeplink(ctx context.Context, event TaskEvent) (string, error) {
	if event.Vault == "" {
		return "", errors.New(ctx, "vault is empty")
	}
	if event.TaskName == "" {
		return "", errors.New(ctx, "task name is empty")
	}
	file := url.PathEscape(event.TasksDir) + "/" + url.PathEscape(event.TaskName+".md")
	return "obsidian://open?vault=" + url.PathEscape(event.Vault) + "&file=" + file, nil
}
