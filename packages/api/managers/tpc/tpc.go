package tpc

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	. "main/packages/onyx/logic"
	. "main/packages/onyx/logic/luau/Api"

	"github.com/crazywolf132/conduit"
	"github.com/crazywolf132/conduit/server"
)

func Server(pipe string) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	tmp := filepath.Join(os.Getenv("APPDATA"), "luna", "sockets")

	cfg := conduit.DefaultServerConfig(fmt.Sprintf("%v/%v.sock", tmp, pipe))
	cfg.MaxMessageSize = 5 * 0x1000000
	cfg.ReadTimeout = time.Hour * 100000
	cfg.WriteTimeout = time.Hour * 100000

	s := server.NewServer(cfg)

	s.Handle("Execute", func(conn *server.Connection, msg *conduit.Message) error {
		defer func() {
			if r := recover(); r != nil {
				Print(3, fmt.Sprintf("%v", r))
			}
		}()

		var task Task
		if err := msg.UnmarshalPayload(&task); err != nil {
			Print(0, "%v", err.Error())
			return err
		}

		switch task.Type {
		case EXECUTE:
			Api.ExecutionChannel.Push(Yieldable{
				Source: Compile(task.Source, CompileOptions{
					OptimizationLevel: 1,
					DebugLevel:        2,
				}),
				Type: Execute,
			})
		}
		return nil
	})
	s.Start()
	defer s.Stop()
	<-ctx.Done()
}
