// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import messaging "github.com/hellofresh/goengine/messaging"
import mock "github.com/stretchr/testify/mock"
import sql "database/sql"

// MessageFactory is an autogenerated mock type for the MessageFactory type
type MessageFactory struct {
	mock.Mock
}

// CreateFromRows provides a mock function with given fields: rows
func (_m *MessageFactory) CreateFromRows(rows *sql.Rows) ([]messaging.Message, error) {
	ret := _m.Called(rows)

	var r0 []messaging.Message
	if rf, ok := ret.Get(0).(func(*sql.Rows) []messaging.Message); ok {
		r0 = rf(rows)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]messaging.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*sql.Rows) error); ok {
		r1 = rf(rows)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
