package lexer

import (
	"math/bits"

	codes "github.com/fredbi/core/json/lexers/error-codes"
)

const (
	maxStack      = 1 << 63
	stackedArray  = 1
	stackedObject = 2
	stackScale    = 63 // 2^8-1
)

func (l *L) isInObject() bool {
	stack := l.nestingLevel[len(l.nestingLevel)-1]

	return stack > 1 && (stack&1 == 0)
}

func (l *L) isInArray() bool {
	stack := l.nestingLevel[len(l.nestingLevel)-1]

	return stack > 1 && (stack&1 == 1)
}

func (l *L) isInContainer() bool {
	if len(l.nestingLevel) > 1 {
		return true
	}

	stack := l.nestingLevel[len(l.nestingLevel)-1]

	return stack > 1
}

// IndentLevel yields the current depth of the container stack.
//
// Example: the following JSON fragment returns 4 tokens, with the
// IndentLevel evolving like so:
//
//	Input: [ { "a": 1 } ]
//	Level: 1 2   22 2 1 0
func (l *L) IndentLevel() int {
	size := len(l.nestingLevel) - 1
	level := size * stackScale
	var stack uint64
	if l.lastStack > 0 {
		stack = l.lastStack
	} else {
		stack = l.nestingLevel[size]
	}
	level += bits.Len64(stack) - 1

	return level
}

// pushObject moves up the stack with an even bit
func (l *L) pushObject() {
	stack := l.nestingLevel[len(l.nestingLevel)-1]

	if l.maxContainerStack > 0 {
		// stack is constrained
		if stack >= maxStack {
			l.nestingLevel = append(l.nestingLevel, stackedObject)
			if len(l.nestingLevel) > l.maxContainerStack {
				l.err = codes.ErrMaxContainerStack
			}

			return
		}

		stack <<= 1
		if bits.Len64(stack) > l.maxContainerStack {
			l.err = codes.ErrMaxContainerStack

			return
		}

		l.nestingLevel[len(l.nestingLevel)-1] = stack

		return
	}

	// unconstrained stack
	if stack >= maxStack {
		l.nestingLevel = append(l.nestingLevel, stackedObject)
	}

	l.nestingLevel[len(l.nestingLevel)-1] = stack << 1
}

// pushArray moves up the stack with an odd bit
func (l *L) pushArray() {
	stack := l.nestingLevel[len(l.nestingLevel)-1]

	if l.maxContainerStack > 0 {
		// stack is constrained
		if stack >= maxStack {
			l.nestingLevel = append(l.nestingLevel, stackedArray)
			if l.maxContainerStack > 0 && len(l.nestingLevel) > l.maxContainerStack {
				l.err = codes.ErrMaxContainerStack
			}

			return
		}

		stack = stack<<1 + 1
		if bits.Len64(stack) > l.maxContainerStack {
			l.err = codes.ErrMaxContainerStack

			return
		}

		l.nestingLevel[len(l.nestingLevel)-1] = stack

		return
	}

	// unconstrained stack
	if stack >= maxStack {
		l.nestingLevel = append(l.nestingLevel, stackedArray)
	}

	l.nestingLevel[len(l.nestingLevel)-1] = stack<<1 + 1
}

// popContainer moves down the stack
func (l *L) popContainer() {
	stack := l.nestingLevel[len(l.nestingLevel)-1]
	if stack <= 1 {
		if len(l.nestingLevel) < 2 {
			panic("dev error: nestingLevel should be initialized with a single element with value 1")
		}

		l.nestingLevel = l.nestingLevel[:len(l.nestingLevel)-1]
		l.popContainer()

		return
	}

	l.nestingLevel[len(l.nestingLevel)-1] = stack >> 1
}
