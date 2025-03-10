// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/JohnPlummer/reddit-client/reddit (interfaces: TestCommentGetter)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	reddit "github.com/JohnPlummer/reddit-client/reddit"
	gomock "github.com/golang/mock/gomock"
)

// MockTestCommentGetter is a mock of TestCommentGetter interface.
type MockTestCommentGetter struct {
	ctrl     *gomock.Controller
	recorder *MockTestCommentGetterMockRecorder
}

// MockTestCommentGetterMockRecorder is the mock recorder for MockTestCommentGetter.
type MockTestCommentGetterMockRecorder struct {
	mock *MockTestCommentGetter
}

// NewMockTestCommentGetter creates a new mock instance.
func NewMockTestCommentGetter(ctrl *gomock.Controller) *MockTestCommentGetter {
	mock := &MockTestCommentGetter{ctrl: ctrl}
	mock.recorder = &MockTestCommentGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTestCommentGetter) EXPECT() *MockTestCommentGetterMockRecorder {
	return m.recorder
}

// SetupComments mocks base method.
func (m *MockTestCommentGetter) SetupComments(arg0 []interface{}) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetupComments", arg0)
}

// SetupComments indicates an expected call of SetupComments.
func (mr *MockTestCommentGetterMockRecorder) SetupComments(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupComments", reflect.TypeOf((*MockTestCommentGetter)(nil).SetupComments), arg0)
}

// SetupCommentsAfter mocks base method.
func (m *MockTestCommentGetter) SetupCommentsAfter(arg0 []reddit.Comment) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetupCommentsAfter", arg0)
}

// SetupCommentsAfter indicates an expected call of SetupCommentsAfter.
func (mr *MockTestCommentGetterMockRecorder) SetupCommentsAfter(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupCommentsAfter", reflect.TypeOf((*MockTestCommentGetter)(nil).SetupCommentsAfter), arg0)
}

// SetupError mocks base method.
func (m *MockTestCommentGetter) SetupError(arg0 error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetupError", arg0)
}

// SetupError indicates an expected call of SetupError.
func (mr *MockTestCommentGetterMockRecorder) SetupError(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetupError", reflect.TypeOf((*MockTestCommentGetter)(nil).SetupError), arg0)
}
