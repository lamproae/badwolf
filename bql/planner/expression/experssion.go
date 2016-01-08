// Copyright 2016 Google Inc. All rights reserved.
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

// Package expression allows to check boolean evaluated expressions against
// a result Table row.
package expression

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/badwolf/bql/table"
)

// Evaluator interface computes the evaluation of a boolean expression.
type Evaluator interface {
	// Evaluate computes the boolean value of the expression given a certain
	// restults table row. It will return an
	// error if it could not be evaluated for the provided table row.
	Evaluate(r table.Row) (bool, error)
}

// OP the operation to be use in the expression evaluation.
type OP int8

const (
	// LT represents '<'
	LT OP = iota
	// GT represents '>''
	GT
	// EQ represents '=''
	EQ
	// NOT represents 'not'
	NOT
	// AND represents 'and'
	AND
	// OR represents 'or'
	OR
)

// String returns a readable string of the operation.
func (o OP) String() string {
	switch o {
	case LT:
		return "<"
	case GT:
		return ">"
	case EQ:
		return "="
	case NOT:
		return "not"
	case AND:
		return "and"
	case OR:
		return "or"
	default:
		return "@UNKNOWN@"
	}
}

// evaluationNode represents the internal representation of one expression.
type evaluationNode struct {
	op OP
	lB string
	rB string
}

// Evaluate the expression.
func (e *evaluationNode) Evaluate(r table.Row) (bool, error) {
	// Binary evaluation
	eval := func() (*table.Cell, *table.Cell, error) {
		var (
			eL, eR *table.Cell
			ok     bool
		)
		eL, ok = r[e.lB]
		if !ok {
			return nil, nil, fmt.Errorf("comparison operations require the binding value for %q for row %q to exist", e.lB, r)
		}
		eR, ok = r[e.rB]
		if !ok {
			return nil, nil, fmt.Errorf("comparison operations require the binding value for %q for row %q to exist", e.rB, r)
		}
		return eL, eR, nil
	}

	cs := func(c *table.Cell) string {
		if c.L != nil {
			return strings.TrimSpace(c.L.ToComparableString())
		}
		return strings.TrimSpace(c.String())
	}

	eL, eR, err := eval()
	if err != nil {
		return false, err
	}
	csEL, csER := cs(eL), cs(eR)
	switch e.op {
	case EQ:
		return reflect.DeepEqual(csEL, csER), nil
	case LT:
		return cs(eL) < cs(eR), nil
	case GT:
		return csEL > csER, nil
	default:
		return false, fmt.Errorf("boolean evaluation require a boolen operation; found %q instead", e.op)
	}
}

// NewEvaluationExpression creates a new evaluator for two bindings in a row.
func NewEvaluationExpression(op OP, lB, rB string) (Evaluator, error) {
	l, r := strings.TrimSpace(lB), strings.TrimSpace(rB)
	if l == "" || r == "" {
		return nil, fmt.Errorf("bindings cannot be empty; got %q, %q", l, r)
	}
	switch op {
	case EQ, LT, GT:
		return &evaluationNode{
			op: op,
			lB: lB,
			rB: rB,
		}, nil
	default:
		return nil, errors.New("evaluation expressions require the operation to be one for the follwing '=', '<', '>'")
	}
}

// booleanNode represents the internal representation of one expression.
type booleanNode struct {
	op OP
	lS bool
	lE Evaluator
	rS bool
	rE Evaluator
}

// Evaluate the expression.
func (e *booleanNode) Evaluate(r table.Row) (bool, error) {
	// Binary evaluation
	eval := func(binary bool) (bool, bool, error) {
		var (
			eL, eR     bool
			errL, errR error
		)
		if !e.lS {
			return false, false, fmt.Errorf("boolean operations require a left operator; found (%q, %q) instead", e.lE, e.rE)
		}
		eL, errL = e.lE.Evaluate(r)
		if errL != nil {
			return false, false, errL
		}
		if binary {
			if !e.rS {
				return false, false, fmt.Errorf("boolean operations require a left operator; found (%q, %q) instead", e.lE, e.rE)
			}
			eR, errR = e.rE.Evaluate(r)
			if errR != nil {
				return false, false, errR
			}
		}
		return eL, eR, nil
	}

	switch e.op {
	case AND:
		eL, eR, err := eval(true)
		if err != nil {
			return false, err
		}
		return eL && eR, nil
	case OR:
		eL, eR, err := eval(true)
		if err != nil {
			return false, err
		}
		return eL || eR, nil
	case NOT:
		eL, _, err := eval(false)
		if err != nil {
			return false, err
		}
		return !eL, nil
	default:
		return false, fmt.Errorf("boolean evaluation require a boolen operation; found %q instead", e.op)
	}
}

// NewBinaryBooleanExpression creates a new binary boolean evalautor.
func NewBinaryBooleanExpression(op OP, lE, rE Evaluator) (Evaluator, error) {
	switch op {
	case AND, OR:
		return &booleanNode{
			op: op,
			lS: true,
			lE: lE,
			rS: true,
			rE: rE,
		}, nil
	default:
		return nil, errors.New("binary boolean expresions require the operation to be one for the follwing 'and', 'or'")
	}
}

// NewUnaryBooleanExpression creates a new unary boolean evalautor.
func NewUnaryBooleanExpression(op OP, lE Evaluator) (Evaluator, error) {
	switch op {
	case NOT:
		return &booleanNode{
			op: op,
			lS: true,
			lE: lE,
			rS: false,
		}, nil
	default:
		return nil, errors.New("unary boolean expresions require the operation to be one for the follwing 'not'")
	}
}