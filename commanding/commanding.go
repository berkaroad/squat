package commanding

import (
	"time"

	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/serialization"
)

const MailCategory string = "command"

type CommandBus interface {
	Send(cmd Command) (*CommandHandleResult, error)
}

type CommandProcess interface {
	Start()
	Stop()
}

type CommandHandleFunc = messaging.MessageHandleFunc[Command]

type Command interface {
	serialization.Serializable
	CommandID() string
	AggregateID() string
	AggregateTypeName() string
}

func NewCommandBase(commandID string, aggregateID string) CommandBase {
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

type CommandHandler messaging.MessageHandler[Command]

type CommandHandlerGroup interface {
	Handlers() map[string]CommandHandler
}

type CommandHandlerProxy messaging.MessageHandlerProxy[Command]

type CommandHandleResult struct {
	FromCommandWatchItem messaging.MessageHandleResultWatchItem
	FromEventWatchItem   messaging.MessageHandleResultWatchItem
}

func (chr *CommandHandleResult) FromCommandHandle(timeout time.Duration) messaging.MessageHandleResult {
	if timeout == 0 {
		timeout = time.Second * 30
	}
	select {
	case fromCommand := <-chr.FromCommandWatchItem.Result():
		return fromCommand
	case <-time.After(timeout):
		return messaging.MessageHandleResult{
			Err: errors.NewWithCode("", "wait command result from command handler timeout"),
		}
	}
}

func (chr *CommandHandleResult) FromEventHandle(timeout time.Duration) messaging.MessageHandleResult {
	if timeout == 0 {
		timeout = time.Second * 30
	}
	waitStartTime := time.Now()
	select {
	case fromCommand := <-chr.FromCommandWatchItem.Result():
		if fromCommand.Err == nil {
			waitDuration := time.Since(waitStartTime)
			select {
			case fromEvent := <-chr.FromEventWatchItem.Result():
				return fromEvent
			case <-time.After(timeout - waitDuration):
				return messaging.MessageHandleResult{
					Code: ErrCodeWaitFromEventHandlerTimeout,
					Err:  ErrWaitFromEventHandlerTimeout,
				}
			}
		} else {
			chr.FromEventWatchItem.Unwatch()
			return fromCommand
		}
	case <-time.After(timeout):
		return messaging.MessageHandleResult{
			Code: ErrCodeWaitFromCommandHandlerTimeout,
			Err:  ErrWaitFromCommandHandlerTimeout,
		}
	}
}

func CreateCommandMail(cmd Command) messaging.Mail[Command] {
	return &commandMail{
		Command: cmd,
	}
}

var _ messaging.Mail[Command] = (*commandMail)(nil)

type commandMail struct {
	Command
}

func (m *commandMail) Metadata() messaging.MessageMetadata {
	return messaging.MessageMetadata{
		ID:                m.Command.CommandID(),
		AggregateID:       m.Command.AggregateID(),
		AggregateTypeName: m.Command.AggregateTypeName(),
	}
}

func (m *commandMail) Unwrap() Command {
	return m.Command
}
