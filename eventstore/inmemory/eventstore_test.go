package inmemory_test

import (
	"context"
	"testing"

	"github.com/hellofresh/goengine/eventstore"
	"github.com/hellofresh/goengine/eventstore/inmemory"
	"github.com/hellofresh/goengine/messaging"
	"github.com/hellofresh/goengine/metadata"
	"github.com/hellofresh/goengine/mocks"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestNewEventStore(t *testing.T) {
	t.Run("Create event store", func(t *testing.T) {
		store := inmemory.NewEventStore(nil)

		assert.IsType(t, (*inmemory.EventStore)(nil), store)
	})

	t.Run("Create event store without logger", func(t *testing.T) {
		store := inmemory.NewEventStore(nil)

		assert.IsType(t, (*inmemory.EventStore)(nil), store)
	})
}

func TestEventStore_Create(t *testing.T) {
	logger, loggerHooks := test.NewNullLogger()
	store := inmemory.NewEventStore(logger)

	err := store.Create("event_stream")

	asserts := assert.New(t)
	asserts.Nil(err)
	asserts.Len(loggerHooks.Entries, 0)

	t.Run("Cannot create a stream twice", func(t *testing.T) {
		err := store.Create("event_stream")

		asserts := assert.New(t)
		asserts.Equal(inmemory.ErrStreamExistsAlready, err)
		asserts.Len(loggerHooks.Entries, 0)
	})
}

func TestEventStore_HasStream(t *testing.T) {
	createThisStream := eventstore.StreamName("my_stream")
	unkownStream := eventstore.StreamName("never_stream")

	logger, loggerHooks := test.NewNullLogger()
	store := inmemory.NewEventStore(logger)

	asserts := assert.New(t)
	asserts.False(store.HasStream(createThisStream))
	asserts.False(store.HasStream(unkownStream))

	store.Create(createThisStream)
	asserts.True(store.HasStream(createThisStream))
	asserts.False(store.HasStream(unkownStream))

	asserts.Len(loggerHooks.Entries, 0)
}

func TestEventStore_Load(t *testing.T) {
	type validTestCase struct {
		title          string
		loadFrom       eventstore.StreamName
		loadCount      *uint
		matcher        metadata.Matcher
		expectedEvents []messaging.Message
	}

	testStreams := map[eventstore.StreamName][]messaging.Message{
		"test": {
			mockMessage(map[string]interface{}{"type": "a", "version": 1}),
			mockMessage(map[string]interface{}{"type": "a", "version": 2}),
			mockMessage(map[string]interface{}{"type": "a", "version": 3}),
			mockMessage(map[string]interface{}{"type": "a", "version": 4}),
			mockMessage(map[string]interface{}{"type": "b", "version": 1}),
		},
		"command": {
			mockMessage(map[string]interface{}{"command": "create"}),
		},
		"empty": {},
	}
	var intTwo uint = 2
	testCases := []validTestCase{
		{
			"Empty event stream",
			"empty",
			nil,
			metadata.NewMatcher(),
			nil,
		},
		{
			"Entire event stream",
			"test",
			nil,
			metadata.NewMatcher(),
			testStreams["test"],
		},
		{
			"All of type a",
			"test",
			nil,
			metadata.WithConstraint(metadata.NewMatcher(), "type", metadata.Equals, "a"),
			testStreams["test"][0:4],
		},
		{
			"Load 2 of type a",
			"test",
			&intTwo,
			metadata.WithConstraint(metadata.NewMatcher(), "type", metadata.Equals, "a"),
			testStreams["test"][0:2],
		},
		{
			"All of type b",
			"test",
			nil,
			metadata.WithConstraint(metadata.NewMatcher(), "type", metadata.Equals, "b"),
			[]messaging.Message{
				testStreams["test"][4],
			},
		},
		{
			"All of type c",
			"test",
			nil,
			metadata.WithConstraint(metadata.NewMatcher(), "type", metadata.Equals, "c"),
			nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			ctx := context.Background()

			logger, loggerHooks := test.NewNullLogger()
			store := inmemory.NewEventStore(logger)

			for stream, events := range testStreams {
				if err := store.Create(stream); !assert.Nil(t, err) {
					t.FailNow()
				}

				if err := store.AppendTo(ctx, stream, events); !assert.Nil(t, err) {
					t.FailNow()
				}
			}

			messages, err := store.Load(ctx, testCase.loadFrom, 1, testCase.loadCount, testCase.matcher)

			asserts := assert.New(t)
			asserts.Equal(testCase.expectedEvents, messages)
			asserts.Nil(err)
			asserts.Len(loggerHooks.Entries, 0)
		})
	}

	t.Run("invalid loads", func(t *testing.T) {
		t.Run("Unknown event stream", func(t *testing.T) {
			ctx := context.Background()
			stream := eventstore.StreamName("unknown")

			store, loggerHooks := createEventStoreWithStream(t, "test")

			messages, err := store.Load(ctx, stream, 1, nil, metadata.NewMatcher())

			asserts := assert.New(t)
			asserts.Equal(inmemory.ErrStreamNotFound, err)
			asserts.Len(messages, 0)
			asserts.Len(loggerHooks.Entries, 0)
		})

		t.Run("incompatible metadata.Matcher", func(t *testing.T) {
			ctx := context.Background()
			stream := eventstore.StreamName("test")
			matcher := metadata.WithConstraint(
				metadata.NewMatcher(),
				"test",
				metadata.GreaterThan,
				true,
			)

			store, loggerHooks := createEventStoreWithStream(t, stream)

			messages, err := store.Load(ctx, stream, 1, nil, matcher)

			asserts := assert.New(t)
			asserts.IsType(inmemory.IncompatibleMatcherError{}, err)
			asserts.Len(messages, 0)
			asserts.Len(loggerHooks.Entries, 0)
		})
	})
}

func TestEventStore_AppendTo(t *testing.T) {
	// For valid appends see TestEventStore_Load

	t.Run("invalid appends", func(t *testing.T) {
		t.Run("Unknown event stream", func(t *testing.T) {
			ctx := context.Background()
			stream := eventstore.StreamName("unknown")

			store, loggerHooks := createEventStoreWithStream(t, "command")

			err := store.AppendTo(ctx, stream, nil)

			asserts := assert.New(t)
			asserts.Equal(inmemory.ErrStreamNotFound, err)
			asserts.Len(loggerHooks.Entries, 0)
		})

		t.Run("Nil message", func(t *testing.T) {
			ctx := context.Background()
			stream := eventstore.StreamName("test")

			store, loggerHooks := createEventStoreWithStream(t, "test")

			err := store.AppendTo(ctx, stream, []messaging.Message{nil})

			asserts := assert.New(t)
			asserts.Equal(inmemory.ErrNilMessage, err)
			asserts.Len(loggerHooks.Entries, 0)
		})

		t.Run("Nil message reference", func(t *testing.T) {
			ctx := context.Background()
			stream := eventstore.StreamName("test")

			store, loggerHooks := createEventStoreWithStream(t, "test")

			err := store.AppendTo(ctx, stream, []messaging.Message{
				(*mocks.Message)(nil),
			})

			asserts := assert.New(t)
			asserts.Equal(inmemory.ErrNilMessage, err)
			asserts.Len(loggerHooks.Entries, 0)
		})
	})
}

func createEventStoreWithStream(t *testing.T, name eventstore.StreamName) (*inmemory.EventStore, *test.Hook) {
	logger, loggerHooks := test.NewNullLogger()
	store := inmemory.NewEventStore(logger)

	err := store.Create(name)
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	return store, loggerHooks
}

func mockMessage(metadataInfo map[string]interface{}) *mocks.Message {
	meta := metadata.New()
	for key, val := range metadataInfo {
		meta = metadata.WithValue(meta, key, val)
	}

	msg := &mocks.Message{}
	msg.On("Metadata").Return(meta)

	return msg
}