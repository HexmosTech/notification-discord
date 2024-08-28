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

// WebhookReq represents the structure of a Discord webhook request
type WebhookReq struct {
	Content string `json:"content"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

// Embed represents a rich embed in a Discord message
type Embed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	// Add more fields as needed
}

// NewWebhookReq creates a new WebhookReq with a simple text message
func NewWebhookReq(content string) *WebhookReq {
	return &WebhookReq{
		Content: content,
	}
}

// AddEmbed adds a new embed to the WebhookReq
func (w *WebhookReq) AddEmbed(title, description string, color int) {
	w.Embeds = append(w.Embeds, Embed{
		Title:       title,
		Description: description,
		Color:       color,
	})
}