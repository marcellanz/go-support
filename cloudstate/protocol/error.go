//
// Copyright 2019 Lightbend Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protocol

/*
Failure is used in two places.
The first place is as part of ClientAction. When sent as a client action, it indicates an error that should be sent to the client - this is an intentional, explicit error message that the user function wants to communicate with the client. In that case, command id is actually irrelevant, since it's already included in a reply message that contains the command id. Typically, this will be used for errors made by the client, for example, commands that failed validation, and could be thought of as an HTTP 4xx error.

The second place that Failure is used is by itself, as a top level message in the oneof for a stream out for the event sourcing, crdt, etc protocols. In this case, it's used for server errors, logically equivalent to HTTP 5xx errors, whose message should not be communicated back to the client (since the client hasn't done anything wrong and can't do anything to rectify, so the error message will be meaningless to the client, plus leaking error messages from arbitrary exceptions can be a security issue), rather a server error status code with a generic error message is sent, but the error will be logged by the proxy for diagnostic purposes. These may or may not have a command id. For example, if the users code throws an unexpected exception, or does something illegal according to the support library, this will be used, and if that occurred during the processing of a command, it will have a command id. However, if the error occurred outside of command handling, it won't have a command id. An example of where we currently send it without a command id is if the proxy sends two init messages (that's not allowed) the user function support library responds with this failure message and no command id. When this is sent, the user function and the proxy are both expected to immediately close the stream after sending/receiving it, since it's not safe to make any assumptions about the state of the stream, the state of the other end of the stream, and any local state related to the stream, when such an error occurs.

Does it make sense to use the same message for both purposes? Maybe, maybe not.
*/

type ClientError struct {
	Err error
}

func (e ClientError) Is(err error) bool {
	_, ok := err.(ClientError)
	return ok
}

func (e ClientError) Error() string {
	return e.Err.Error()
}

func (e ClientError) Unwrap() error {
	return e.Err
}

type ServerError struct {
	Failure *Failure
	Err     error
}

func (e ServerError) Is(err error) bool {
	_, ok := err.(ServerError)
	return ok
}

func (e ServerError) Error() string {
	return e.Err.Error()
}

func (e ServerError) Unwrap() error {
	return e.Err
}
