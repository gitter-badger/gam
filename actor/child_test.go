package actor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type CreateChildMessage struct{}
type GetChildCountRequest struct{}
type GetChildCountResponse struct{ ChildCount int }
type CreateChildActor struct{}

func (*CreateChildActor) Receive(context Context) {
	switch context.Message().(type) {
	case CreateChildMessage:
		context.Spawn(FromProducer(NewBlackHoleActor))
	case GetChildCountRequest:
		reply := GetChildCountResponse{ChildCount: len(context.Children())}
		context.Respond(reply)
	}
}

func NewCreateChildActor() Actor {
	return &CreateChildActor{}
}

func TestActorCanCreateChildren(t *testing.T) {
	actor := Spawn(FromProducer(NewCreateChildActor))
	defer actor.Stop()
	expected := 10
	for i := 0; i < expected; i++ {
		actor.Tell(CreateChildMessage{})
	}
	response, err := actor.RequestFuture(GetChildCountRequest{}, testTimeout).Result()
	if err != nil {
		assert.Fail(t, "timed out")
		return
	}
	assert.Equal(t, expected, response.(GetChildCountResponse).ChildCount)
}

type CreateChildThenStopActor struct {
	replyTo *PID
}

type GetChildCountMessage2 struct {
	ReplyDirectly  *PID
	ReplyAfterStop *PID
}

func (state *CreateChildThenStopActor) Receive(context Context) {
	switch msg := context.Message().(type) {
	case CreateChildMessage:
		context.Spawn(FromProducer(NewBlackHoleActor))
	case GetChildCountMessage2:
		msg.ReplyDirectly.Tell(true)
		state.replyTo = msg.ReplyAfterStop
	case *Stopped:
		reply := GetChildCountResponse{ChildCount: len(context.Children())}
		state.replyTo.Tell(reply)
	}
}

func NewCreateChildThenStopActor() Actor {
	return &CreateChildThenStopActor{}
}

func TestActorCanStopChildren(t *testing.T) {

	actor := Spawn(FromProducer(NewCreateChildThenStopActor))
	count := 10
	for i := 0; i < count; i++ {
		actor.Tell(CreateChildMessage{})
	}

	future := NewFuture(testTimeout)
	future2 := NewFuture(testTimeout)
	actor.Tell(GetChildCountMessage2{ReplyDirectly: future.PID(), ReplyAfterStop: future2.PID()})

	//wait for the actor to reply to the first responsePID
	err := future.Wait()
	if err != nil {
		assert.Fail(t, "timed out")
		return
	}

	//then send a stop command
	actor.Stop()

	//wait for the actor to stop and get the result from the stopped handler
	response, err := future2.Result()
	if err != nil {
		assert.Fail(t, "timed out")
		return
	}
	//we should have 0 children when the actor is stopped
	assert.Equal(t, 0, response.(GetChildCountResponse).ChildCount)
}
