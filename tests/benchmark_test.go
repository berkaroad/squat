package tests

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/berkaroad/squat/caching"
	"github.com/berkaroad/squat/commanding"
	"github.com/berkaroad/squat/commanding/inmemorycb"
	"github.com/berkaroad/squat/errors"
	"github.com/berkaroad/squat/eventing"
	"github.com/berkaroad/squat/eventing/inmemoryeb"
	"github.com/berkaroad/squat/eventsourcing"
	"github.com/berkaroad/squat/messaging"
	"github.com/berkaroad/squat/messaging/inmemorymhrn"
	"github.com/berkaroad/squat/serialization"
	"github.com/berkaroad/squat/store/eventstore"
	"github.com/berkaroad/squat/store/eventstore/inmemoryes"
	"github.com/berkaroad/squat/store/publishedstore"
	"github.com/berkaroad/squat/store/publishedstore/inmemoryps"
	"github.com/berkaroad/squat/store/snapshotstore"
	"github.com/berkaroad/squat/store/snapshotstore/inmemoryss"
)

func BenchmarkAccount(b *testing.B) {
	ctx := context.TODO()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelWarn,
	}))
	slog.SetDefault(logger)

	var mhrn = inmemorymhrn.Default()

	var cd commanding.CommandDispatcher = (&commanding.DefaultCommandDispatcher{}).
		Initialize(&messaging.DefaultMailboxProvider[commanding.CommandData]{
			MailboxCapacity:    1000,
			GetMailboxName:     func(aggregateID, aggregateTypeName string) string { return aggregateID },
			AutoReleaseTimeout: 30 * time.Second,
		}, mhrn)
	var cb commanding.CommandBus = inmemorycb.Default().Initialize(cd, mhrn)
	var commandprocessor commanding.CommandProcessor = cb.(commanding.CommandProcessor)

	var ed eventing.EventDispatcher = (&eventing.DefaultEventDispatcher{}).
		Initialize(&messaging.DefaultMailboxProvider[eventing.EventData]{
			MailboxCapacity:    1000,
			GetMailboxName:     func(aggregateID, aggregateTypeName string) string { return aggregateID },
			AutoReleaseTimeout: 30 * time.Second,
		}, mhrn)
	var eb eventing.EventBus = inmemoryeb.Default().Initialize(ed)
	var eventprocessor eventing.EventProcessor = eb.(eventing.EventProcessor)

	var es = inmemoryes.Default()
	var ess = (&eventstore.DefaultEventStoreSaver{}).Initialize(es)
	var ps = inmemoryps.Default()
	var pss = (&publishedstore.DefaultPublishedStoreSaver{}).Initialize(ps)
	var ss = inmemoryss.Default()
	var sss = (&snapshotstore.DefaultSnapshotStoreSaver{}).Initialize(ss)

	// business initialization
	serialization.Map[AccountSnapshot]()
	serialization.Map[CreateAccountCommand]()
	serialization.Map[DepositCommand]()
	serialization.Map[WithdrawCommand]()
	serialization.Map[RemoveAccountCommand]()
	serialization.Map[AccountCreated]()
	serialization.Map[AccountBalanceChanged]()
	serialization.Map[AccountRemoved]()
	cd.SubscribeMulti(&AccountCommandHandler{
		Repo: (&eventsourcing.EventSourcedRepository[*Account]{
			SnapshotEnabled: true,
			CacheEnabled:    true,
			CacheExpiration: time.Minute * 2,
		}).Initialize(eb, es, ess, ps, pss, ss, sss, (&caching.MemoryCache{CleanInterval: time.Second * 30}).Initialize(serialization.DefaultBinary()), serialization.DefaultText(), serialization.DefaultBinary()),
	})
	ed.SubscribeMulti(&AccountViewGenerator{})
	ed.AddProxy(&IdempotentEventHandlerProxy{})

	ess.Start()
	pss.Start()
	sss.Start()
	eventprocessor.Start()
	commandprocessor.Start()

	b.Cleanup(func() {
		logger.Info("Stopping...")
		commandprocessor.Stop()
		eventprocessor.Stop()
		sss.Stop()
		pss.Stop()
		ess.Stop()
		logger.Info("Stopped")
	})

	processCmdResult := func(wg *sync.WaitGroup, cmd commanding.Command, timeout time.Duration) {
		defer wg.Done()

		cmdResult, err := cb.Execute(ctx, cmd)
		if err != nil {
			logger.Error(err.Error())
		}
		fromCommandResult := cmdResult.FromEventHandle(ctx, timeout)
		if fromCommandResult.Err == nil {
			return
		}
		if fromCommandResult.Err != nil {
			logger.Error(fromCommandResult.Err.Error(),
				slog.String("error-code", errors.GetErrorCode(fromCommandResult.Err)),
				slog.String("command-id", cmd.CommandID()),
				slog.String("command-type", cmd.TypeName()),
				slog.String("aggregate-id", cmd.AggregateID()),
			)
		}
	}

	b.Run("Creation", func(b *testing.B) {
		if b.N == 1 {
			// skip warmup
			return
		}

		wg := &sync.WaitGroup{}
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			cmd := &CreateAccountCommand{
				CommandBase: commanding.NewCommandBase(NewUUID(), fmt.Sprintf("acc-%d", i)),
				Name:        fmt.Sprintf("Account %d", i),
			}
			go processCmdResult(wg, cmd, time.Second*10)
		}
		wg.Wait()
	})

	b.Run("Disposit", func(b *testing.B) {
		if b.N == 1 {
			// skip warmup
			return
		}

		wg := &sync.WaitGroup{}
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			cmd := &DepositCommand{
				CommandBase: commanding.NewCommandBase(NewUUID(), fmt.Sprintf("acc-%d", i)),
				Amount:      1.1,
			}
			go processCmdResult(wg, cmd, time.Second*10)
		}
		wg.Wait()
	})

	b.Run("Withdraw", func(b *testing.B) {
		if b.N == 1 {
			// skip warmup
			return
		}

		wg := &sync.WaitGroup{}
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			cmd := &WithdrawCommand{
				CommandBase: commanding.NewCommandBase(NewUUID(), fmt.Sprintf("acc-%d", i)),
				Amount:      1.1,
			}
			go processCmdResult(wg, cmd, time.Second*10)
		}
		wg.Wait()
	})

	b.Run("Remove", func(b *testing.B) {
		if b.N == 1 {
			// skip warmup
			return
		}

		wg := &sync.WaitGroup{}
		for i := 0; i < b.N; i++ {
			wg.Add(1)
			cmd := &RemoveAccountCommand{
				CommandBase: commanding.NewCommandBase(NewUUID(), fmt.Sprintf("acc-%d", i)),
			}
			go processCmdResult(wg, cmd, time.Second*10)
		}
		wg.Wait()
	})
}
