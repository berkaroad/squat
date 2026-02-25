package commanding

import (
	"context"
	"strings"

	"github.com/berkaroad/squat/domain"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/serialization"
)

const MailCategory string = "command"

type Command interface {
	serialization.Serializable
	CommandID() string
	AggregateID() string
	AggregateTypeName() string
}

type CommandBus interface {
	Send(ctx context.Context, cmd Command) error
	Execute(ctx context.Context, cmd Command) (*CommandHandleResult, error)
}

type CommandProcess interface {
	Start()
	Stop()
}

type CommandData struct {
	Command
	Extensions map[string]string
}

func (data *CommandData) SetCustomExtension(ctx context.Context, key string, val string) {
	metadata := messaging.FromContext(ctx)
	if metadata != nil && metadata.Category == MailCategory && !strings.HasPrefix(key, messaging.SysExtensionKeyPrefix) {
		metadata.Extensions = metadata.Extensions.Clone().
			Set(messaging.ExtensionKey(key), val)
		data.Extensions = metadata.Extensions
	}
}

type CommandHandleFunc = messaging.MessageHandleFunc[CommandData]

type CommandHandler messaging.MessageHandler[CommandData]

type CommandHandlerGroup interface {
	CommandHandlers() map[string]CommandHandler
}

type CommandHandlerProxy messaging.MessageHandlerProxy[CommandData]

func NewCommandBase(commandID string, aggregateID string) CommandBase {
	if commandID == "" {
		panic(ErrEmptyCommandID)
	}
	if aggregateID == "" {
		panic(domain.ErrEmptyAggregateID)
	}
	return CommandBase{
		C_ID:          commandID,
		C_AggregateID: aggregateID,
	}
}

var _ Command = CommandBase{}

// Base of command, should override method 'TypeName()' and 'AggregateTypeName()'
type CommandBase struct {
	C_ID          string
	C_AggregateID string
}

func (c CommandBase) TypeName() string {
	panic("method 'TypeName()' not impletement")
}
func (c CommandBase) CommandID() string {
	return c.C_ID
}
func (c CommandBase) AggregateID() string {
	return c.C_AggregateID
}
func (c CommandBase) AggregateTypeName() string {
	panic("method 'AggregateTypeName()' not impletement")
}

func CreateCommandMail(data *CommandData) messaging.Mail[CommandData] {
	return &commandMail{
		Command:     data.Command,
		commandData: data,
	}
}

var _ messaging.Mail[CommandData] = (*commandMail)(nil)

type commandMail struct {
	Command
	commandData *CommandData
}

func (m *commandMail) Metadata() messaging.MessageMetadata {
	return messaging.MessageMetadata{
		MessageID:     m.Command.CommandID(),
		MessageType:   m.Command.TypeName(),
		AggregateID:   m.Command.AggregateID(),
		AggregateType: m.Command.AggregateTypeName(),
		Category:      MailCategory,
		Extensions:    m.commandData.Extensions,
	}
}

func (m *commandMail) Unwrap() CommandData {
	return *m.commandData
}
