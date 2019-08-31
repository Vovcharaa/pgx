package pgproto3

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Frontend acts as a client for the PostgreSQL wire protocol version 3.
type Frontend struct {
	cr ChunkReader
	w  io.Writer

	// Backend message flyweights
	authentication       Authentication
	backendKeyData       BackendKeyData
	bindComplete         BindComplete
	closeComplete        CloseComplete
	commandComplete      CommandComplete
	copyBothResponse     CopyBothResponse
	copyData             CopyData
	copyInResponse       CopyInResponse
	copyOutResponse      CopyOutResponse
	copyDone             CopyDone
	dataRow              DataRow
	emptyQueryResponse   EmptyQueryResponse
	errorResponse        ErrorResponse
	functionCallResponse FunctionCallResponse
	noData               NoData
	noticeResponse       NoticeResponse
	notificationResponse NotificationResponse
	parameterDescription ParameterDescription
	parameterStatus      ParameterStatus
	parseComplete        ParseComplete
	readyForQuery        ReadyForQuery
	rowDescription       RowDescription
	portalSuspended      PortalSuspended

	bodyLen    int
	msgType    byte
	partialMsg bool
}

// NewFrontend creates a new Frontend.
func NewFrontend(cr ChunkReader, w io.Writer) (*Frontend, error) {
	return &Frontend{cr: cr, w: w}, nil
}

// Send sends a message to the backend.
func (b *Frontend) Send(msg FrontendMessage) error {
	_, err := b.w.Write(msg.Encode(nil))
	return err
}

// Receive receives a message from the backend.
func (b *Frontend) Receive() (BackendMessage, error) {
	if !b.partialMsg {
		header, err := b.cr.Next(5)
		if err != nil {
			return nil, err
		}

		b.msgType = header[0]
		b.bodyLen = int(binary.BigEndian.Uint32(header[1:])) - 4
		b.partialMsg = true
	}

	var msg BackendMessage
	switch b.msgType {
	case '1':
		msg = &b.parseComplete
	case '2':
		msg = &b.bindComplete
	case '3':
		msg = &b.closeComplete
	case 'A':
		msg = &b.notificationResponse
	case 'c':
		msg = &b.copyDone
	case 'C':
		msg = &b.commandComplete
	case 'd':
		msg = &b.copyData
	case 'D':
		msg = &b.dataRow
	case 'E':
		msg = &b.errorResponse
	case 'G':
		msg = &b.copyInResponse
	case 'H':
		msg = &b.copyOutResponse
	case 'I':
		msg = &b.emptyQueryResponse
	case 'K':
		msg = &b.backendKeyData
	case 'n':
		msg = &b.noData
	case 'N':
		msg = &b.noticeResponse
	case 'R':
		msg = &b.authentication
	case 's':
		msg = &b.portalSuspended
	case 'S':
		msg = &b.parameterStatus
	case 't':
		msg = &b.parameterDescription
	case 'T':
		msg = &b.rowDescription
	case 'V':
		msg = &b.functionCallResponse
	case 'W':
		msg = &b.copyBothResponse
	case 'Z':
		msg = &b.readyForQuery
	default:
		return nil, fmt.Errorf("unknown message type: %c", b.msgType)
	}

	msgBody, err := b.cr.Next(b.bodyLen)
	if err != nil {
		return nil, err
	}

	b.partialMsg = false

	err = msg.Decode(msgBody)
	return msg, err
}
