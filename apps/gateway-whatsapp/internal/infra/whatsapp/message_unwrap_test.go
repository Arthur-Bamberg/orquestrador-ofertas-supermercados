package whatsapp_test

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/gateway-whatsapp/internal/infra/whatsapp"
)

func TestUnwrapMessage_ephemeralWrappingDeviceSent(t *testing.T) {
	inner := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("tomate e banana"),
		},
	}
	deviceSent := &waE2E.Message{
		DeviceSentMessage: &waE2E.DeviceSentMessage{
			DestinationJID: proto.String("120363abc@g.us"),
			Message:        inner,
		},
	}
	ephemeral := &waE2E.Message{
		EphemeralMessage: &waE2E.FutureProofMessage{
			Message: deviceSent,
		},
	}

	unwrapped := whatsapp.UnwrapMessage(ephemeral)
	if unwrapped == nil || unwrapped.GetExtendedTextMessage().GetText() != "tomate e banana" {
		t.Fatalf("esperava 'tomate e banana', obteve: %+v", unwrapped)
	}

	texto := whatsapp.TextoDe(ephemeral)
	if texto != "tomate e banana" {
		t.Fatalf("TextoDe esperava 'tomate e banana', obteve: %q", texto)
	}
}

func TestUnwrapMessage_deviceSentWrappingEphemeral(t *testing.T) {
	inner := &waE2E.Message{
		Conversation: proto.String("leite desnatado"),
	}
	ephemeral := &waE2E.Message{
		EphemeralMessage: &waE2E.FutureProofMessage{
			Message: inner,
		},
	}
	deviceSent := &waE2E.Message{
		DeviceSentMessage: &waE2E.DeviceSentMessage{
			DestinationJID: proto.String("120363abc@g.us"),
			Message:        ephemeral,
		},
	}

	texto := whatsapp.TextoDe(deviceSent)
	if texto != "leite desnatado" {
		t.Fatalf("TextoDe esperava 'leite desnatado', obteve: %q", texto)
	}
}

func TestUnwrapMessage_groupMentioned(t *testing.T) {
	inner := &waE2E.Message{
		Conversation: proto.String("cenoura"),
	}
	mentioned := &waE2E.Message{
		GroupMentionedMessage: &waE2E.FutureProofMessage{
			Message: inner,
		},
	}

	texto := whatsapp.TextoDe(mentioned)
	if texto != "cenoura" {
		t.Fatalf("TextoDe esperava 'cenoura', obteve: %q", texto)
	}
}
