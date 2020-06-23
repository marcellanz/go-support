package eventsourced

import (
	"errors"
	"fmt"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

// sendFailure sends a given error to the proxy.
// see: https://github.com/cloudstateio/cloudstate/pull/119#discussion_r375619440
func sendFailure(e error, server protocol.EventSourced_HandleServer) error {
	pf := &protocol.ProtocolFailure{}
	if errors.As(e, &pf) {
		desc := pf.F.Description
		if desc == "" {
			if e := errors.Unwrap(e); e != nil {
				desc = e.Error()
			}
		}
		//err := server.Send(&protocol.EventSourcedStreamOut{
		//	Message: &protocol.EventSourcedStreamOut_Failure{
		//		Failure: f,
		//	},
		//})
		err := server.Send(&protocol.EventSourcedStreamOut{
			Message: &protocol.EventSourcedStreamOut_Reply{
				Reply: &protocol.EventSourcedReply{
					CommandId: pf.F.CommandId,
					ClientAction: &protocol.ClientAction{
						Action: &protocol.ClientAction_Failure{
							Failure: &protocol.Failure{
								CommandId:   pf.F.CommandId,
								Description: desc,
							},
						},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("send of EventSourcedStreamOut Failure failed with:%v, %w", err, e)
		}
		return nil
	}
	cf := &protocol.ClientFailure{}
	if is := errors.As(e, &cf); is {
		desc := cf.F.Description
		if desc == "" {
			if e := errors.Unwrap(e); e != nil {
				desc = e.Error()
			}
		}
		err := server.Send(&protocol.EventSourcedStreamOut{
			Message: &protocol.EventSourcedStreamOut_Reply{
				Reply: &protocol.EventSourcedReply{
					CommandId: cf.F.CommandId,
					ClientAction: &protocol.ClientAction{
						Action: &protocol.ClientAction_Failure{
							Failure: &protocol.Failure{
								CommandId:   cf.F.CommandId,
								Description: desc,
							},
						},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("send of EventSourcedStreamOut Failure failed with:%v, %w", err, e)
		}
		return nil
	}
	// any other failure is sent as a protocol failure
	err := server.Send(&protocol.EventSourcedStreamOut{
		Message: &protocol.EventSourcedStreamOut_Failure{
			Failure: &protocol.Failure{
				Description: e.Error(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send of EventSourcedStreamOut Failure failed with:%v, %w", err, e)
	}
	return nil
}
