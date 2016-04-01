package model

import (
	aperrors "gateway/errors"
	apsql "gateway/sql"

	"github.com/jmoiron/sqlx/types"
)

type PushMessage struct {
	AccountID        int64 `json:"-"`
	UserID           int64 `json:"-"`
	APIID            int64 `json:"-" path:"apiID"`
	RemoteEndpointID int64 `json:"-" path:"endpointID"`
	PushChannelID    int64 `json:"-" path:"pushChannelID"`

	ID           int64          `json:"id,omitempty" path:"id"`
	PushDeviceID int64          `json:"push_device_id" db:"push_device_id" path:"pushDeviceID"`
	Stamp        int64          `json:"stamp"`
	Data         types.JsonText `json:"-" db:"data"`
}

func (d *PushMessage) Validate(isInsert bool) aperrors.Errors {
	errors := make(aperrors.Errors)
	return errors
}

func (d *PushMessage) ValidateFromDatabaseError(err error) aperrors.Errors {
	errors := make(aperrors.Errors)
	return errors
}

func (m *PushMessage) All(db *apsql.DB) ([]*PushMessage, error) {
	messages := []*PushMessage{}
	err := db.Select(&messages, db.SQL("push_messages/all"),
		m.PushDeviceID, m.PushChannelID, m.RemoteEndpointID, m.APIID, m.AccountID)
	if err != nil {
		return nil, err
	}
	for _, message := range messages {
		message.AccountID = m.AccountID
		message.UserID = m.UserID
		message.APIID = m.APIID
		message.RemoteEndpointID = m.RemoteEndpointID
		message.PushChannelID = m.PushChannelID
	}
	return messages, nil
}

func (m *PushMessage) Find(db *apsql.DB) (*PushMessage, error) {
	message := PushMessage{
		AccountID:        m.AccountID,
		UserID:           m.UserID,
		APIID:            m.APIID,
		RemoteEndpointID: m.RemoteEndpointID,
		PushChannelID:    m.PushChannelID,
	}
	err := db.Get(&message, db.SQL("push_messages/find"), m.ID,
		m.PushDeviceID, m.PushChannelID, m.RemoteEndpointID, m.APIID, m.AccountID)
	return &message, err
}

func (m *PushMessage) Delete(tx *apsql.Tx) error {
	err := tx.DeleteOne(tx.SQL("push_messages/delete"), m.ID,
		m.PushDeviceID, m.PushChannelID, m.RemoteEndpointID, m.APIID, m.AccountID)
	if err != nil {
		return err
	}
	return tx.Notify("push_messages", m.AccountID, m.UserID, m.APIID, 0, m.ID, apsql.Delete)
}

func (m *PushMessage) Insert(tx *apsql.Tx) error {
	data, err := marshaledForStorage(m.Data)
	if err != nil {
		return err
	}

	m.ID, err = tx.InsertOne(tx.SQL("push_messages/insert"),
		m.PushDeviceID, m.PushChannelID, m.RemoteEndpointID, m.APIID, m.AccountID,
		m.Stamp, data)
	if err != nil {
		return err
	}
	return tx.Notify("push_messages", m.AccountID, m.UserID, m.APIID, 0, m.ID, apsql.Insert)
}

func (m *PushMessage) Update(tx *apsql.Tx) error {
	data, err := marshaledForStorage(m.Data)
	if err != nil {
		return err
	}

	err = tx.UpdateOne(tx.SQL("push_messages/update"), m.Stamp, data, m.ID,
		m.PushDeviceID, m.PushChannelID, m.RemoteEndpointID, m.APIID, m.AccountID)
	if err != nil {
		return err
	}
	return tx.Notify("push_messages", m.AccountID, m.UserID, m.APIID, 0, m.ID, apsql.Update)
}
