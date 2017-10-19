package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereumproject/go-ethereum/rpc"
)

const errorMsg = "client returned error"
const errorCode = 123

type fakeClient struct {
	returnError bool
	recvError   bool
}

func (f *fakeClient) SupportedModules() (map[string]string, error) {
	if f.returnError {
		return nil, errors.New(errorMsg)
	}
	return map[string]string{
		"validModule": "1.0",
		"otherModule": "2.0",
		"thirdModule": "1.1",
		"lastModule":  "1.0",
	}, nil
}

func (f *fakeClient) Send(req interface{}) error {
	// noop
	if f.returnError {
		return errors.New(errorMsg)
	}
	return nil
}

func (f *fakeClient) Recv(msg interface{}) error {
	// noop
	if f.returnError {
		return errors.New(errorMsg)
	}

	switch msg.(type) {
	case *rpc.JSONResponse:
		if f.recvError {
			msg.(*rpc.JSONResponse).Error = &rpc.JSONError{Code: errorCode, Message: errorMsg}
		} else {
			msg.(*rpc.JSONResponse).Result = "fake result"
		}
	}

	return nil
}

func (f *fakeClient) Close() {
	// noop
}

func TestValidateArguments(t *testing.T) {
	var testCases = []struct {
		input    []string // input arguments
		expected string   // error message or ""
	}{
		{[]string{"validModule", "someMethod"}, ""},
		{[]string{"validModule", "someMethod", "arg"}, ""},
		{[]string{"validModule", "someMethod", "arg1", "arg2"}, ""},
		{[]string{"thirdModule", "someMethod"}, ""},
		{[]string{"invalidModule", "someMethod"}, "unknown API module: invalidModule"},
		{[]string{}, "api command requires at least 2 arguments (module and method), 0 provided"},
		{[]string{"validModule"}, "api command requires at least 2 arguments (module and method), 1 provided"},
	}

	client := &fakeClient{false, false}

	for _, tc := range testCases {
		t.Run(strings.Join(tc.input, ","), func(t *testing.T) {
			ctx := createContext(tc.input)
			err := validateArguments(ctx, client)
			if err != nil {
				if tc.expected != err.Error() {
					t.Fatalf("Expected error: '%s', got '%s'", tc.expected, err.Error())
				}
			} else {
				if tc.expected != "" {
					t.Fatal("Expected no error")
				}
			}
		})
	}
}

func TestValidateArgumentsClientError(t *testing.T) {
	client := &fakeClient{true, false}
	ctx := createContext([]string{"validModule", "method", "arg"})
	err := validateArguments(ctx, client)
	if err == nil || errorMsg != err.Error() {
		t.Fatalf("Expected client returned error!")
	}
}

func TestCallRPC(t *testing.T) {
	client := &fakeClient{false, false}
	ctx := createContext([]string{"validModule", "method", "arg"})

	res, err := callRPC(ctx, client)
	if err != nil {
		t.Fatal("Expected no error, got: ", err)
	}

	if res == nil {
		t.Fatal("Expected result")
	}
}

func TestCallRPCError(t *testing.T) {
	client := &fakeClient{true, false}
	ctx := createContext([]string{"validModule", "method", "arg"})

	res, err := callRPC(ctx, client)
	if err == nil || errorMsg != err.Error() {
		t.Fatal("Expected '%s', got '%s': ", errorMsg, err.Error())
	}

	if res != nil {
		t.Fatal("Expected no result")
	}

}

func TestCallRPCMethodError(t *testing.T) {
	client := &fakeClient{false, true}
	ctx := createContext([]string{"validModule", "method", "arg"})

	res, err := callRPC(ctx, client)
	if err == nil {
		t.Fatal("Expected error")
	}

	if res != nil {
		t.Fatal("Expected no result")
	}

	expected := fmt.Sprintf("error in validModule_method: %s (code: %d)", errorMsg, errorCode)
	if expected != err.Error() {
		t.Fail()
	}
}

func createContext(args []string) *cli.Context {
	var flags flag.FlagSet
	flags.Parse(args)
	return cli.NewContext(nil, &flags, nil)
}
