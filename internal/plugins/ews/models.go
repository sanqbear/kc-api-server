package ews

import "time"

// EmailListItem represents a summary of an email for list views
type EmailListItem struct {
	ItemID         string    `json:"item_id"`
	ConversationID string    `json:"conversation_id,omitempty"`
	Subject        string    `json:"subject"`
	From           string    `json:"from"`
	FromEmail      string    `json:"from_email"`
	ReceivedDate   time.Time `json:"received_date"`
	HasAttachments bool      `json:"has_attachments"`
	IsRead         bool      `json:"is_read"`
	Preview        string    `json:"preview,omitempty"`
}

// EmailDetail represents full details of an email
type EmailDetail struct {
	ItemID            string         `json:"item_id"`
	ConversationID    string         `json:"conversation_id,omitempty"`
	Subject           string         `json:"subject"`
	Body              string         `json:"body"`
	BodyType          string         `json:"body_type"` // "Text" or "HTML"
	From              EmailAddress   `json:"from"`
	ToRecipients      []EmailAddress `json:"to_recipients"`
	CcRecipients      []EmailAddress `json:"cc_recipients,omitempty"`
	BccRecipients     []EmailAddress `json:"bcc_recipients,omitempty"`
	ReceivedDate      time.Time      `json:"received_date"`
	SentDate          time.Time      `json:"sent_date"`
	HasAttachments    bool           `json:"has_attachments"`
	IsRead            bool           `json:"is_read"`
	Importance        string         `json:"importance,omitempty"`
	Categories        []string       `json:"categories,omitempty"`
	InternetMessageID string         `json:"internet_message_id,omitempty"`
}

// EmailAddress represents an email address with display name
type EmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// EmailThread represents a conversation thread
type EmailThread struct {
	ConversationID string          `json:"conversation_id"`
	Topic          string          `json:"topic"`
	Emails         []EmailListItem `json:"emails"`
}

// ListEmailsRequest represents the request to list emails
type ListEmailsRequest struct {
	Mailbox    string `json:"mailbox" validate:"required,email"`
	FolderName string `json:"folder_name"` // "Inbox", "SentItems", etc. Default: "Inbox"
	Limit      int    `json:"limit"`       // Default: 50, Max: 100
	Offset     int    `json:"offset"`      // For pagination
}

// ListEmailsResponse represents the response with email list
type ListEmailsResponse struct {
	Emails []EmailListItem `json:"emails"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// GetEmailDetailRequest represents the request to get email details
type GetEmailDetailRequest struct {
	Mailbox string `json:"mailbox" validate:"required,email"`
	ItemID  string `json:"item_id" validate:"required"`
}

// GetEmailDetailResponse represents the response with email details and thread
type GetEmailDetailResponse struct {
	Email  EmailDetail     `json:"email"`
	Thread []EmailListItem `json:"thread,omitempty"`
}

// SOAP Envelope structures for EWS XML communication

// SOAPEnvelope is the root element for SOAP requests
type SOAPEnvelope struct {
	XMLName struct{} `xml:"soap:Envelope"`
	XMLNS   string   `xml:"xmlns:soap,attr"`
	XSI     string   `xml:"xmlns:xsi,attr"`
	M       string   `xml:"xmlns:m,attr"`
	T       string   `xml:"xmlns:t,attr"`
	Header  SOAPHeader
	Body    SOAPBody
}

// SOAPHeader contains SOAP header elements
type SOAPHeader struct {
	XMLName               struct{} `xml:"soap:Header"`
	RequestServerVersion  RequestServerVersion
	ExchangeImpersonation *ExchangeImpersonation `xml:",omitempty"`
}

// RequestServerVersion specifies the EWS API version
type RequestServerVersion struct {
	XMLName struct{} `xml:"t:RequestServerVersion"`
	Version string   `xml:"Version,attr"`
}

// ExchangeImpersonation allows accessing another mailbox
type ExchangeImpersonation struct {
	XMLName       struct{} `xml:"t:ExchangeImpersonation"`
	ConnectingSID ConnectingSID
}

// ConnectingSID identifies the account to impersonate
type ConnectingSID struct {
	XMLName            struct{} `xml:"t:ConnectingSID"`
	PrimarySmtpAddress string   `xml:"t:PrimarySmtpAddress"`
}

// SOAPBody contains the SOAP body with the request
type SOAPBody struct {
	XMLName interface{} `xml:"soap:Body"`
	Content interface{}
}

// FindItemRequest is used to search for items (emails) in a folder
type FindItemRequest struct {
	XMLName         struct{} `xml:"m:FindItem"`
	Traversal       string   `xml:"Traversal,attr"`
	ItemShape       ItemShape
	ParentFolderIds ParentFolderIds
}

// ItemShape defines what properties to return
type ItemShape struct {
	XMLName              struct{} `xml:"m:ItemShape"`
	BaseShape            string   `xml:"t:BaseShape"`
	AdditionalProperties *AdditionalProperties `xml:",omitempty"`
}

// AdditionalProperties specifies additional fields to retrieve
type AdditionalProperties struct {
	XMLName  struct{} `xml:"t:AdditionalProperties"`
	FieldURI []FieldURI
}

// FieldURI identifies a specific property
type FieldURI struct {
	XMLName  struct{} `xml:"t:FieldURI"`
	FieldURI string   `xml:"FieldURI,attr"`
}

// ParentFolderIds specifies which folder to search
type ParentFolderIds struct {
	XMLName               struct{} `xml:"m:ParentFolderIds"`
	DistinguishedFolderId DistinguishedFolderId
}

// DistinguishedFolderId identifies a well-known folder
type DistinguishedFolderId struct {
	XMLName struct{} `xml:"t:DistinguishedFolderId"`
	Id      string   `xml:"Id,attr"`
	Mailbox *Mailbox `xml:",omitempty"`
}

// Mailbox identifies the mailbox
type Mailbox struct {
	XMLName      struct{} `xml:"t:Mailbox"`
	EmailAddress string   `xml:"t:EmailAddress"`
}

// GetItemRequest is used to get full details of specific items
type GetItemRequest struct {
	XMLName   struct{} `xml:"m:GetItem"`
	ItemShape ItemShape
	ItemIds   ItemIds
}

// ItemIds contains the IDs of items to retrieve
type ItemIds struct {
	XMLName struct{} `xml:"m:ItemIds"`
	ItemId  []ItemId
}

// ItemId identifies a specific item
type ItemId struct {
	XMLName   struct{} `xml:"t:ItemId"`
	Id        string   `xml:"Id,attr"`
	ChangeKey string   `xml:"ChangeKey,attr,omitempty"`
}

// FindItemResponse represents the response from FindItem
type FindItemResponse struct {
	XMLName struct{} `xml:"Envelope"`
	Body    FindItemResponseBody
}

// FindItemResponseBody contains the response body
type FindItemResponseBody struct {
	XMLName          struct{} `xml:"Body"`
	FindItemResponse FindItemResponseMessage
}

// FindItemResponseMessage contains the actual response
type FindItemResponseMessage struct {
	XMLName          struct{} `xml:"FindItemResponse"`
	ResponseMessages ResponseMessages
}

// ResponseMessages contains response message elements
type ResponseMessages struct {
	XMLName                 struct{} `xml:"ResponseMessages"`
	FindItemResponseMessage FindItemResponseMessageType
}

// FindItemResponseMessageType contains the items found
type FindItemResponseMessageType struct {
	XMLName       struct{} `xml:"FindItemResponseMessage"`
	ResponseClass string   `xml:"ResponseClass,attr"`
	ResponseCode  string   `xml:"ResponseCode"`
	RootFolder    RootFolder
}

// RootFolder contains the search results
type RootFolder struct {
	XMLName                 struct{} `xml:"RootFolder"`
	TotalItemsInView        int      `xml:"TotalItemsInView,attr"`
	IncludesLastItemInRange bool     `xml:"IncludesLastItemInRange,attr"`
	Items                   Items
}

// Items contains the list of items
type Items struct {
	XMLName struct{} `xml:"Items"`
	Message []Message
}

// Message represents an email message
type Message struct {
	XMLName          struct{}              `xml:"Message"`
	ItemId           MessageItemId         `xml:"ItemId"`
	Subject          string                `xml:"Subject"`
	DateTimeReceived string                `xml:"DateTimeReceived"`
	DateTimeSent     string                `xml:"DateTimeSent"`
	From             MessageEmailAddress   `xml:"From"`
	IsRead           bool                  `xml:"IsRead"`
	HasAttachments   bool                  `xml:"HasAttachments"`
	ConversationId   MessageConversationId `xml:"ConversationId"`
	Body             *MessageBody          `xml:"Body,omitempty"`
	ToRecipients     *ToRecipients         `xml:"ToRecipients,omitempty"`
	CcRecipients     *CcRecipients         `xml:"CcRecipients,omitempty"`
	Importance       string                `xml:"Importance,omitempty"`
	Categories       *Categories           `xml:"Categories,omitempty"`
	InternetMessageId string               `xml:"InternetMessageId,omitempty"`
}

// MessageItemId contains item identification
type MessageItemId struct {
	XMLName   struct{} `xml:"ItemId"`
	Id        string   `xml:"Id,attr"`
	ChangeKey string   `xml:"ChangeKey,attr"`
}

// MessageConversationId identifies the conversation
type MessageConversationId struct {
	XMLName struct{} `xml:"ConversationId"`
	Id      string   `xml:"Id,attr"`
}

// MessageEmailAddress represents the From field
type MessageEmailAddress struct {
	XMLName struct{} `xml:"From"`
	Mailbox MessageMailbox
}

// MessageMailbox contains email address details
type MessageMailbox struct {
	XMLName      struct{} `xml:"Mailbox"`
	Name         string   `xml:"Name"`
	EmailAddress string   `xml:"EmailAddress"`
}

// MessageBody contains the email body
type MessageBody struct {
	XMLName  struct{} `xml:"Body"`
	BodyType string   `xml:"BodyType,attr"`
	Content  string   `xml:",chardata"`
}

// ToRecipients contains To recipients
type ToRecipients struct {
	XMLName struct{} `xml:"ToRecipients"`
	Mailbox []MessageMailbox
}

// CcRecipients contains CC recipients
type CcRecipients struct {
	XMLName struct{} `xml:"CcRecipients"`
	Mailbox []MessageMailbox
}

// Categories contains email categories
type Categories struct {
	XMLName struct{} `xml:"Categories"`
	String  []string `xml:"String"`
}

// GetItemResponse represents the response from GetItem
type GetItemResponse struct {
	XMLName struct{} `xml:"Envelope"`
	Body    GetItemResponseBody
}

// GetItemResponseBody contains the response body
type GetItemResponseBody struct {
	XMLName         struct{} `xml:"Body"`
	GetItemResponse GetItemResponseMessage
}

// GetItemResponseMessage contains the actual response
type GetItemResponseMessage struct {
	XMLName          struct{} `xml:"GetItemResponse"`
	ResponseMessages GetItemResponseMessages
}

// GetItemResponseMessages contains response messages
type GetItemResponseMessages struct {
	XMLName                struct{} `xml:"ResponseMessages"`
	GetItemResponseMessage GetItemResponseMessageType
}

// GetItemResponseMessageType contains the items
type GetItemResponseMessageType struct {
	XMLName       struct{} `xml:"GetItemResponseMessage"`
	ResponseClass string   `xml:"ResponseClass,attr"`
	ResponseCode  string   `xml:"ResponseCode"`
	Items         Items
}
