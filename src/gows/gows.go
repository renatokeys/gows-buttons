package gows

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/devlikeapro/gows/storage"
	"github.com/devlikeapro/gows/storage/sqlstorage"
	_ "github.com/jackc/pgx/v5"     // Import the Postgres driver
	_ "github.com/mattn/go-sqlite3" // Import the SQLite driver
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// GoWS it's Go WebSocket or WhatSapp ;)
type GoWS struct {
	*whatsmeow.Client
	int     *whatsmeow.DangerousInternalClient
	Context context.Context
	Storage *storage.Storage

	events              chan interface{}
	eventsMu            sync.RWMutex
	eventsClosed        bool
	cancelContext       context.CancelFunc
	container           *sqlstorage.GContainer
	storageEventHandler *StorageEventHandler
}

func (gows *GoWS) reissueEvent(event interface{}) {
	// Handle all panic and log error + stack
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			gows.Log.Errorf("Panic happened in reissue event: %v. Stack: %s. Event: %v", err, stack, event)
		}
	}()

	var data interface{}
	switch event.(type) {
	case *events.Connected:
		// Populate the ConnectedEventData with the ID and PushName
		data = &ConnectedEventData{
			ID:       gows.Store.ID,
			LID:      &gows.Store.LID,
			PushName: gows.Store.PushName,
		}

	case *events.Message:
		data = event
		if event.(*events.Message).Message.GetEncEventResponseMessage() != nil {
			go gows.handleEncEventResponse(gows.Context, event.(*events.Message))
		} else if event.(*events.Message).Message.GetPollUpdateMessage() != nil {
			go gows.handleEncPollVote(gows.Context, event.(*events.Message))
		}

	default:
		data = event
	}

	gows.emitEvent(data)
}

func (gows *GoWS) handleEvent(event interface{}) {
	go gows.reissueEvent(event)
	go gows.storageEventHandler.handleEvent(event)
}

func (gows *GoWS) Start() error {
	gows.AddEventHandler(gows.handleEvent)

	// Not connected, listen for QR code events
	if gows.Store.ID == nil {
		gows.listenQRCodeEvents()
	}

	return gows.Connect()
}

func (gows *GoWS) listenQRCodeEvents() {
	// No ID stored, new login
	qrChan, _ := gows.GetQRChannel(gows.Context)

	// reissue from QrChan to events
	go func() {
		for {
			select {
			case <-gows.Context.Done():
				return
			case qr := <-qrChan:
				// If the event is empty, we should stop the goroutine
				if qr.Event == "" {
					return
				}
				gows.emitEvent(qr)
			}
		}
	}()
}

func (gows *GoWS) Stop() {
	gows.cancelContext()
	gows.Disconnect()
	err := gows.container.Close()
	if err != nil {
		gows.Log.Errorf("Error closing container: %v", err)
	}
	gows.eventsMu.Lock()
	if !gows.eventsClosed {
		close(gows.events)
		gows.eventsClosed = true
	}
	gows.eventsMu.Unlock()
}

func (gows *GoWS) GetOwnId() types.JID {
	if gows == nil {
		return types.EmptyJID
	}
	id := gows.Store.ID
	if id == nil {
		return types.EmptyJID
	}
	return *id
}

func BuildSession(
	ctx context.Context,
	log waLog.Logger,
	dialect string,
	address string,
	ignoreJids *IgnoreJidsConfig,
) (*GoWS, error) {
	// Prepare the database
	container, err := sqlstorage.New(dialect, address, log.Sub("Database"))
	if err != nil {
		return nil, err
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return nil, err
	}

	// Configure the client
	client := whatsmeow.NewClient(deviceStore, log.Sub("Client"))
	client.AutomaticMessageRerequestFromPhone = true
	client.EmitAppStateEventsOnFullSync = true
	client.InitialAutoReconnect = true

	ctx, cancel := context.WithCancel(ctx)
	gows := &GoWS{
		client,
		client.DangerousInternals(),
		ctx,
		nil,
		make(chan interface{}, 10),
		sync.RWMutex{},
		false,
		cancel,
		container,
		nil,
	}
	gows.Storage = BuildStorage(container, gows)
	gows.storageEventHandler = &StorageEventHandler{
		gows:       gows,
		log:        gows.Log.Sub("Storage"),
		storage:    gows.Storage,
		ignoreJids: ignoreJids,
	}
	gows.GetMessageForRetry = gows.storageEventHandler.GetMessageForRetry
	return gows, nil
}

func (gows *GoWS) GetEventChannel() <-chan interface{} {
	return gows.events
}

func (gows *GoWS) emitEvent(data interface{}) {
	gows.eventsMu.RLock()
	defer gows.eventsMu.RUnlock()

	if gows.eventsClosed {
		return
	}

	select {
	case <-gows.Context.Done():
		return
	case gows.events <- data:
	}
}

func (gows *GoWS) SendMessage(ctx context.Context, to types.JID, msg *waE2E.Message, extra whatsmeow.SendRequestExtra) (message *events.Message, err error) {
	var resp whatsmeow.SendResponse

	if to.User == "status" && to.Server == types.BroadcastServer {
		// Broadcast messages (Status)
		result, err := gows.SendStatusMessage(ctx, to, msg, extra)
		if err != nil {
			return nil, err
		}
		resp = *result
	} else {
		resp, err = gows.Client.SendMessage(ctx, to, msg, extra)
		if err != nil {
			return nil, err
		}
	}

	info := &types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     to,
			Sender:   gows.GetOwnId(),
			IsFromMe: true,
			IsGroup:  to.Server == types.GroupServer,
		},
		ID:        resp.ID,
		Timestamp: resp.Timestamp,
		ServerID:  resp.ServerID,
	}
	evt := &events.Message{Info: *info, Message: msg, RawMessage: msg}
	go gows.handleEvent(evt)
	return evt, nil
}

// MarkRead marks messages as read and emits a receipt event
func (gows *GoWS) MarkRead(ids []types.MessageID, chat types.JID, sender types.JID, receiptType types.ReceiptType) error {
	timestamp := time.Now()
	err := gows.Client.MarkRead(ids, timestamp, chat, sender, receiptType)
	if err != nil {
		return err
	}

	receipt := &events.Receipt{
		MessageSource: types.MessageSource{
			Chat:   chat,
			Sender: sender,
		},
		MessageIDs: ids,
		Type:       receiptType,
		Timestamp:  timestamp,
	}
	go gows.handleEvent(receipt)
	return nil
}
