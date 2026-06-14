package core

type Event interface {
	String() string // 使用字符串描述 Event， 用于调试和基础的信息显示
}

type OutStream[E Event] chan E
