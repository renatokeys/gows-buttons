package gows

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"strings"
)

func HasNewsletterSuffix(s string) bool {
	return strings.HasSuffix(s, "@"+types.NewsletterServer)
}

func IsNewsletter(jid types.JID) bool {
	return jid.Server == types.NewsletterServer
}

type GetNewsletterMessagesByInviteParams struct {
	Count int
}

type NewsletterMessagesResp struct {
	NewsletterJID types.JID
	Messages      []*types.NewsletterMessage
}

// GetNewsletterMessagesByInvite gets messages in a WhatsApp channel using an invite code.
func (cli *GoWS) GetNewsletterMessagesByInvite(code string, params *GetNewsletterMessagesByInviteParams) (*NewsletterMessagesResp, error) {
	key := strings.TrimPrefix(code, whatsmeow.NewsletterLinkPrefix)
	attrs := waBinary.Attrs{
		"type":  "invite",
		"key":   key,
		"count": 100,
	}
	if params != nil {
		if params.Count != 0 {
			attrs["count"] = params.Count
		}
	}

	resp, err := cli.int.SendIQ(whatsmeow.DangerousInfoQuery{
		Namespace: "newsletter",
		Type:      "get",
		To:        types.ServerJID,
		Content: []waBinary.Node{{
			Tag:   "messages",
			Attrs: attrs,
		}},
		Context: context.TODO(),
	})
	if err != nil {
		return nil, err
	}
	messagesNode, ok := resp.GetOptionalChildByTag("messages")
	if !ok {
		return nil, &whatsmeow.ElementMissingError{Tag: "messages", In: "newsletter messages response"}
	}
	messages := cli.int.ParseNewsletterMessages(&messagesNode)
	messages = filterMessageNull(messages)
	jid, ok := messagesNode.Attrs["jid"].(types.JID)
	if !ok {
		return nil, fmt.Errorf("no jid in messages response")
	}
	response := NewsletterMessagesResp{
		NewsletterJID: jid,
		Messages:      messages,
	}
	return &response, nil
}

func filterMessageNull(messages []*types.NewsletterMessage) []*types.NewsletterMessage {
	var filtered []*types.NewsletterMessage
	for _, message := range messages {
		if message.Message != nil {
			filtered = append(filtered, message)
		}
	}
	return filtered
}

type SearchPageParams struct {
	Count       int
	StartCursor string
}

type SearchNewsletterByViewParams struct {
	Page       SearchPageParams
	View       string
	Categories []string
	Countries  []string
}

type SearchNewsletterByTextParams struct {
	Page       SearchPageParams
	Text       string
	Categories []string
}

type SearchPageResult struct {
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
}

type SearchNewsletterResult struct {
	Page        SearchPageResult
	Newsletters []*types.NewsletterMetadata
}

const (
	queryNewslettersDirectoryList   = "6190824427689257"
	queryNewslettersDirectorySearch = "6802402206520139"
)

type Map map[string]interface{}

type respSearchNewsletter struct {
	PageInfo    *SearchPageResult           `json:"page_info"`
	Newsletters []*types.NewsletterMetadata `json:"result"`
}

type respSearchNewsletterDirectoryList struct {
	Data *respSearchNewsletter `json:"xwa2_newsletters_directory_list"`
}
type respSearchNewsletterDirectorySearch struct {
	Data *respSearchNewsletter `json:"xwa2_newsletters_directory_search"`
}

// SearchNewsletterByView searches for WhatsApp channels by views.
func (cli *GoWS) SearchNewsletterByView(query SearchNewsletterByViewParams) (*SearchNewsletterResult, error) {
	variables := Map{
		"input": Map{
			"view": query.View,
			"filters": Map{
				"country_codes": query.Countries,
				"categories":    query.Categories,
			},
			"limit":        query.Page.Count,
			"start_cursor": query.Page.StartCursor,
		},
	}
	data, err := cli.int.SendMexIQ(context.TODO(), queryNewslettersDirectoryList, variables)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("no data returned")
	}

	var respData respSearchNewsletterDirectoryList
	err = json.Unmarshal(data, &respData)
	if err != nil {
		return nil, err
	}
	result := SearchNewsletterResult{
		Page:        *respData.Data.PageInfo,
		Newsletters: respData.Data.Newsletters,
	}
	return &result, nil
}

// SearchNewsletterByText searches for WhatsApp channels by text.
func (cli *GoWS) SearchNewsletterByText(query SearchNewsletterByTextParams) (*SearchNewsletterResult, error) {
	variables := Map{
		"input": Map{
			"search_text":  query.Text,
			"categories":   query.Categories,
			"limit":        query.Page.Count,
			"start_cursor": query.Page.StartCursor,
		},
	}
	data, err := cli.int.SendMexIQ(context.TODO(), queryNewslettersDirectorySearch, variables)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("no data returned")
	}

	var respData respSearchNewsletterDirectorySearch
	err = json.Unmarshal(data, &respData)
	if err != nil {
		return nil, err
	}
	result := SearchNewsletterResult{
		Page:        *respData.Data.PageInfo,
		Newsletters: respData.Data.Newsletters,
	}
	return &result, nil
}
