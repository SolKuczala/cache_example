package main

import (
	"cacheservice/proto"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type commandSet struct {
	cmd      string
	key      string
	oldValue string
	newValue string
}

// Arguments passed to the cli.
const (
	serverAddr      = "localhost:5555"
	cmdParam        = 1
	keyParam        = 2
	firstValueParam = 3
	newValueParam   = 4

	usage = `
	Usage: client <command> [arguments]
	
	The commands are:

	set		set the key along the value, i.e.: <set> [key] [value]
	get		get the value from the provided key, i.e.: <get> [key]
	cmpAndSet	cmpAndSet provided the key along the old value, set the new value, i.e.:
			cmpAndSet <key> [oldvalue] [newValue]
`
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := startClient(); err != nil {
		switch {
		case errors.Is(err, errCommandIncorrect):
			fmt.Println(usage)
		case errors.Is(err, errCommandNotProvided):
			fmt.Println(usage)
		case errors.Is(err, errKeyOrValueMissing):
			fmt.Println(usage)
		default:
			log.Fatal(err)
		}
	}
}

// startClient parse input from user and sends commands with values to the server
// returns error if something fails.
func startClient() error {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("could not connect to the server %w", err)
	}
	defer conn.Close()
	// initialize a cache with executeCommands in a go routine, waiting to receive commands
	cacheClient := proto.NewCacheServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// will wait until an abort signal is made from the user
	go func() {
		sigsCh := make(chan os.Signal, 1)
		signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigsCh)
		select {
		case <-ctx.Done():
			return
		case <-sigsCh:
			fmt.Println("\n aborting command...")
		}
		cancel()
	}()

	cmdsSet, err := getCommandSet()
	if err != nil {
		return fmt.Errorf("startClient: getCommandset: %w", err)
	}
	if err = sendCommandWithParams(ctx, cmdsSet, cacheClient); err != nil {
		return fmt.Errorf("startClient: %w", err)
	}
	return nil
}

// getCommandSet get the set of params provided by the user and returns the set of commands.
// If no command is passed to it, an error is returned. If filled with the wrong command,
// an empty set is returned.
func getCommandSet() (commandSet, error) {
	var cs commandSet
	if len(os.Args) < 2 {
		return cs, errCommandNotProvided
	}
	switch {
	case os.Args[cmdParam] == "get" && len(os.Args) == 3:
		cs.cmd = "get"
		cs.key = os.Args[keyParam]

	case os.Args[cmdParam] == "set" && len(os.Args) == 4:
		cs.cmd = "set"
		cs.key = os.Args[keyParam]
		cs.newValue = os.Args[firstValueParam]

	case os.Args[cmdParam] == "cmpAndSet" && len(os.Args) == 5:
		cs.cmd = "cmpAndSet"
		cs.key = os.Args[keyParam]
		cs.oldValue = os.Args[firstValueParam]
		cs.newValue = os.Args[newValueParam]

	default:
		return cs, nil
	}
	return cs, nil
}

// sendCommandWithParams receives a cacheClient connection with flags struct which contains:
// command, key, value
// to be executed by the grpc server.
func sendCommandWithParams(ctx context.Context, cs commandSet, cacheClient proto.CacheServiceClient) error {
	switch {
	case cs.cmd == "set":
		if cs.key == "" || cs.newValue == "" {
			return fmt.Errorf("sendCommandWithParams: %w", errKeyOrValueMissing)
		}
		_, err := cacheClient.Set(ctx, &proto.SetRequest{Key: cs.key, Value: cs.newValue})
		if err != nil {
			return fmt.Errorf("sendCommandWithParams Set: %w", err)
		}
		fmt.Println("set it successfully")
		return nil

	case cs.cmd == "get":
		result, err := cacheClient.Get(ctx, &proto.GetRequest{Key: cs.key})
		if err != nil {
			if statusCode, ok := status.FromError(err); ok {
				if statusCode.Code() == codes.NotFound {
					return err
				}
			}
			return fmt.Errorf("sendCommandWithParams Get: %w", err)
		}
		switch result.GetValue() {
		case "":
			fmt.Println("value is empty")
		default:
			fmt.Println("value from get:", result.GetValue())
		}

	case cs.cmd == "cmpAndSet":
		if cs.key == "" || cs.oldValue == "" || cs.newValue == "" {
			return fmt.Errorf("sendCommandWithParams: cmpAndSet: %w", errKeyOrValueMissing)
		}
		resp, err := cacheClient.CmpAndSet(ctx, &proto.CmpAndSetRequest{
			Key:      cs.key,
			OldValue: cs.oldValue,
			NewValue: cs.newValue,
		})
		if err != nil {
			return fmt.Errorf("sendCommandWithParams: Set: %w", err)
		}

		if resp.GetChanged() {
			fmt.Println("value successfully changed")
			return nil
		}
		fmt.Println("command did not change")
	default:
		return fmt.Errorf("sendCommandWithParams: %w", errCommandIncorrect)
	}
	return nil
}

/*TODO: return better errors to the user
 */
