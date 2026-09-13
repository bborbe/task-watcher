// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package publish turns a matched task event into a notification-publish
// command and sends it into the shared notification core.
package publish

// TaskEvent is one matched task change, carrying everything the notification
// message and its metadata need.
type TaskEvent struct {
	Vault    string
	TasksDir string
	TaskName string
	Status   string
	Phase    string
	Assignee string
}
